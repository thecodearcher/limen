package twofactor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/thecodearcher/aegis"
)

type challengePayload struct {
	UserID any    `json:"user_id"`
	Exp    int64  `json:"exp"`
	Type   string `json:"type"`
}

type twoFactorPlugin struct {
	core            *aegis.AegisCore
	httpCore        *aegis.AegisHTTPCore
	cookies         *aegis.CookieManager
	twoFactorSchema *twoFactorSchema
	userSchema      *userWithTwoFactorSchema
	config          *config
	totp            *totp
	otp             *otp
	backupCodes     *backupCodes
}

func (t *twoFactorPlugin) Name() aegis.PluginName {
	return aegis.PluginTwoFactor
}

func New(opts ...ConfigOption) *twoFactorPlugin {
	config := &config{
		secret:           getTOTPSecret(),
		totp:             NewDefaultTOTPConfig(),
		otp:              NewDefaultOTPConfig(),
		cookieExpiration: defaultChallengeExpiration,
		cookieName:       defaultChallengeCookieName,
	}
	for _, opt := range opts {
		opt(config)
	}

	return &twoFactorPlugin{
		config: config,
	}
}

func (t *twoFactorPlugin) Initialize(core *aegis.AegisCore) error {
	t.core = core
	t.cookies = core.Cookies()
	if len(t.config.secret) == 0 {
		if base := core.SigningSecret(); len(base) > 0 {
			t.config.secret = base
		}
	}
	if len(t.config.secret) == 0 {
		return fmt.Errorf("two-factor requires a secret: set twofactor.WithSecret, Config.SigningSecret, or AEGIS_TOTP_SECRET / AEGIS_SECRET")
	}
	t.totp = newDefaultTOTP(t, t.config.totp)
	t.backupCodes = newBackupCodes(t, t.config.backupCodes)
	if t.config.otp.enabled {
		t.otp = newDefaultOTP(t, t.config.otp)
	}
	return nil
}

func (t *twoFactorPlugin) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		BasePath:   "/two-factor",
		Middleware: []aegis.Middleware{},
		Hooks: &aegis.Hooks{
			After: []*aegis.Hook{
				{
					PathMatcher: func(ctx *aegis.HookContext) bool {
						return ctx.RouteID() == "signin"
					},
					Run: t.handleSigninHook,
				},
			},
		},
	}
}

func (t *twoFactorPlugin) handleSigninHook(ctx *aegis.HookContext) bool {
	original := ctx.GetResponse()
	if original == nil || original.IsError || original.StatusCode != http.StatusOK {
		return true
	}

	authResult := ctx.GetAuthResult()
	if authResult == nil || authResult.User == nil {
		return true
	}

	rawUser := authResult.User.Raw()
	twoFactorEnabled, ok := rawUser[t.userSchema.GetTwoFactorEnabledField()].(bool)
	if !ok || !twoFactorEnabled {
		return true
	}

	t.revokeSessionFromResponse(ctx)
	challengeToken, err := t.generateChallengeToken(authResult.User.ID)
	if err != nil {
		ctx.ModifyResponse(http.StatusInternalServerError, err)
		return false
	}

	t.setChallengeCookie(ctx, challengeToken)

	ctx.ModifyResponse(http.StatusOK, map[string]any{
		"two_factor_required": true,
	})

	return true
}

func (t *twoFactorPlugin) revokeSessionFromResponse(ctx *aegis.HookContext) {
	if t.httpCore == nil {
		return
	}

	sessionCookieName := t.httpCore.SessionCookieName()
	if sessionCookieName == "" {
		return
	}

	sessionToken := aegis.ExtractCookieValue(ctx.Response().Header(), sessionCookieName)
	if sessionToken != "" {
		_ = t.core.SessionManager.RevokeSession(ctx.Request().Context(), sessionToken)
	}
	ctx.RemoveResponseCookie(sessionCookieName)
	t.httpCore.Cookies().ClearSessionCookie(ctx.Response())
}

func (t *twoFactorPlugin) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	t.httpCore = httpCore
	handlers := newTwoFactorHandlers(t, httpCore.Responder, httpCore)

	// Global endpoints
	routeBuilder.ProtectedPOST("/initiate-setup", "two-factor-initiate-setup", handlers.InitiateTwoFactorSetup)
	routeBuilder.ProtectedPOST("/finalize-setup", "two-factor-finalize-setup", handlers.FinalizeTwoFactorSetup)
	routeBuilder.ProtectedPOST("/disable", "two-factor-disable", handlers.Disable)
	routeBuilder.POST("/verify-login", "two-factor-verify-login", handlers.VerifyLoginWithTwoFactor)

	t.totp.registerRoutes(httpCore, routeBuilder)
	t.backupCodes.registerRoutes(httpCore, routeBuilder)

	if t.config.otp.enabled {
		t.otp.registerRoutes(httpCore, routeBuilder)
	}
}

func (t *twoFactorPlugin) GetSchemas(schema *aegis.SchemaConfig) []aegis.SchemaIntrospector {
	twoFactorSchema := newDefaultTwoFactorSchema()
	userWithTwoFactorSchema := newDefaultSchemaUserTwoFactor(schema.User)
	t.userSchema = userWithTwoFactorSchema
	t.twoFactorSchema = twoFactorSchema

	userWithTwoFactorExtension := aegis.NewSchemaDefinitionForExtension(
		aegis.CoreSchemaUsers,
		userWithTwoFactorSchema,
		aegis.WithSchemaField(string(UserWithTwoFactorSchemaEnabledField), aegis.ColumnTypeBool, aegis.WithDefaultValue("false"), aegis.WithNullable(false)),
	)

	twoFactorTable := aegis.NewSchemaDefinitionForTable(
		aegis.SchemaName(TwoFactorSchemaTableName),
		TwoFactorSchemaTableName,
		twoFactorSchema,
		aegis.WithSchemaIDField(schema),
		aegis.WithSchemaField(string(TwoFactorSchemaUserIDField), schema.GetIDColumnType()),
		aegis.WithSchemaField(string(TwoFactorSchemaSecretField), aegis.ColumnTypeString, aegis.WithNullable(true)),
		aegis.WithSchemaField(string(TwoFactorSchemaBackupCodesField), aegis.ColumnTypeText, aegis.WithNullable(true)),
		aegis.WithSchemaForeignKey(aegis.ForeignKeyDefinition{
			Name:             "fk_two_factors_users_user_id",
			Column:           TwoFactorSchemaUserIDField,
			ReferencedSchema: aegis.CoreSchemaUsers,
			ReferencedField:  aegis.SchemaIDField,
			OnDelete:         aegis.FKActionCascade,
			OnUpdate:         aegis.FKActionCascade,
		}),
		aegis.WithSchemaUniqueIndex("idx_two_factors_user_id", []aegis.SchemaField{TwoFactorSchemaUserIDField}),
	)

	return []aegis.SchemaIntrospector{
		userWithTwoFactorExtension,
		twoFactorTable,
	}
}

func (t *twoFactorPlugin) FindTwoFactorByUserID(ctx context.Context, userID any) (*TwoFactor, error) {
	twoFactor, err := t.core.FindOne(ctx, t.twoFactorSchema, []aegis.Where{
		aegis.Eq(t.twoFactorSchema.GetUserIDField(), userID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return twoFactor.(*TwoFactor), nil
}

func (t *twoFactorPlugin) CreateTwoFactor(ctx context.Context, userID any, encryptedSecret string, encryptedBackupCodes string) error {
	twoFactor := &TwoFactor{
		UserID:      userID,
		Secret:      encryptedSecret,
		BackupCodes: encryptedBackupCodes,
	}
	return t.core.Create(ctx, t.twoFactorSchema, twoFactor, nil)
}

func (t *twoFactorPlugin) DeleteTwoFactor(ctx context.Context, userID any) error {
	return t.core.Delete(ctx, t.twoFactorSchema, []aegis.Where{
		aegis.Eq(t.twoFactorSchema.GetUserIDField(), userID),
	})
}

func (t *twoFactorPlugin) encrypt(value string) (string, error) {
	return aegis.EncryptXChaCha(value, t.config.secret, nil)
}

func (t *twoFactorPlugin) decrypt(secret string) (string, error) {
	return aegis.DecryptXChaCha(secret, t.config.secret, nil)
}

func (t *twoFactorPlugin) generateChallengeToken(userID any) (string, error) {
	payload := challengePayload{
		UserID: userID,
		Exp:    time.Now().Add(t.config.cookieExpiration).Unix(),
		Type:   challengeTokenType,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return t.encrypt(string(jsonPayload))
}

func (t *twoFactorPlugin) verifyChallengeToken(token string) (*challengePayload, error) {
	decrypted, err := t.decrypt(token)
	if err != nil {
		return nil, ErrInvalidChallenge
	}

	var payload challengePayload
	if err := json.Unmarshal([]byte(decrypted), &payload); err != nil {
		return nil, ErrInvalidChallenge
	}

	if payload.Type != challengeTokenType {
		return nil, ErrInvalidChallenge
	}

	if time.Now().Unix() > payload.Exp {
		return nil, ErrChallengeExpired
	}

	return &payload, nil
}

func (t *twoFactorPlugin) setChallengeCookie(ctx *aegis.HookContext, token string) {
	t.cookies.SetOnHookCtx(ctx, t.config.cookieName, token, int(t.config.cookieExpiration.Seconds()))
}

func (t *twoFactorPlugin) clearChallengeCookie(w http.ResponseWriter) {
	t.cookies.Delete(w, t.config.cookieName)
}

func (t *twoFactorPlugin) getChallengeFromCookie(r *http.Request) (string, error) {
	val, err := t.cookies.Get(r, t.config.cookieName)
	if err != nil {
		return "", ErrChallengeMissing
	}
	return val, nil
}
