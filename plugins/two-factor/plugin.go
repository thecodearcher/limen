package twofactor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/thecodearcher/limen"
)

type challengePayload struct {
	UserID any    `json:"user_id"`
	Exp    int64  `json:"exp"`
	Type   string `json:"type"`
}

type twoFactorPlugin struct {
	core            *limen.LimenCore
	httpCore        *limen.LimenHTTPCore
	twoFactorSchema *twoFactorSchema
	userSchema      *userWithTwoFactorSchema
	config          *config
	totp            *totp
	otp             *otp
	backupCodes     *backupCodes
}

func (t *twoFactorPlugin) Name() limen.PluginName {
	return limen.PluginTwoFactor
}

func New(opts ...ConfigOption) *twoFactorPlugin {
	config := &config{
		secret:                           getTOTPSecret(),
		totp:                             NewDefaultTOTPConfig(),
		otp:                              NewDefaultOTPConfig(),
		backupCodes:                      NewDefaultBackupCodesConfig(),
		cookieExpiration:                 defaultChallengeExpiration,
		cookieName:                       defaultChallengeCookieName,
		revokeOtherSessionsOnStateChange: true,
	}
	for _, opt := range opts {
		opt(config)
	}

	return &twoFactorPlugin{
		config: config,
	}
}

func (t *twoFactorPlugin) Initialize(core *limen.LimenCore) error {
	t.core = core
	if len(t.config.secret) == 0 {
		if base := core.Secret(); len(base) > 0 {
			t.config.secret = base
		}
	}
	if len(t.config.secret) == 0 {
		return fmt.Errorf("two-factor requires a secret: set twofactor.WithSecret, Config.Secret, or LIMEN_TOTP_SECRET / LIMEN_SECRET")
	}
	t.totp = newDefaultTOTP(t, t.config.totp)
	t.backupCodes = newBackupCodes(t, t.config.backupCodes)
	if t.config.otp.enabled {
		t.otp = newDefaultOTP(t, t.config.otp)
	}
	return nil
}

func (t *twoFactorPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath:   "/two-factor",
		Middleware: []limen.Middleware{},
		Hooks: &limen.Hooks{
			After: []*limen.Hook{
				{
					PathMatcher: func(ctx *limen.HookContext) bool {
						return ctx.RouteID() == "signin"
					},
					Run: t.handleSigninHook,
				},
			},
		},
	}
}

func (t *twoFactorPlugin) handleSigninHook(ctx *limen.HookContext) bool {
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

func (t *twoFactorPlugin) revokeSessionFromResponse(ctx *limen.HookContext) {
	if t.httpCore == nil {
		return
	}

	sessionCookieName := t.httpCore.SessionCookieName()
	if sessionCookieName == "" {
		return
	}

	sessionToken := limen.ExtractCookieValue(ctx.Response().Header(), sessionCookieName)
	if sessionToken != "" {
		_ = t.core.SessionManager.RevokeSession(ctx.Request().Context(), sessionToken)
	}
	ctx.RemoveResponseCookie(sessionCookieName)
	t.core.Cookies().ClearSessionCookie(ctx.Response())
}

func (t *twoFactorPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	t.httpCore = httpCore
	handlers := newTwoFactorHandlers(t, httpCore.Responder, httpCore)

	// Global endpoints
	routeBuilder.ProtectedPOST("/initiate-setup", "two-factor-initiate-setup", handlers.InitiateTwoFactorSetup)
	routeBuilder.ProtectedPOST("/finalize-setup", "two-factor-finalize-setup", handlers.FinalizeTwoFactorSetup)
	routeBuilder.ProtectedPOST("/disable", "two-factor-disable", handlers.Disable)
	routeBuilder.POST("/verify", "two-factor-verify", handlers.VerifyLoginWithTwoFactor)

	t.totp.registerRoutes(httpCore, routeBuilder)
	t.backupCodes.registerRoutes(httpCore, routeBuilder)

	if t.config.otp.enabled {
		t.otp.registerRoutes(httpCore, routeBuilder)
	}
}

func (t *twoFactorPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	twoFactorSchema := newDefaultTwoFactorSchema()
	userWithTwoFactorSchema := newDefaultSchemaUserTwoFactor(schema.User)
	t.userSchema = userWithTwoFactorSchema
	t.twoFactorSchema = twoFactorSchema

	userWithTwoFactorExtension := limen.NewSchemaDefinitionForExtension(
		limen.CoreSchemaUsers,
		userWithTwoFactorSchema,
		limen.WithSchemaField(string(UserWithTwoFactorSchemaEnabledField), limen.ColumnTypeBool, limen.WithDefaultValue("false"), limen.WithNullable(false)),
	)

	twoFactorTable := limen.NewSchemaDefinitionForTable(
		limen.SchemaName(TwoFactorSchemaTableName),
		TwoFactorSchemaTableName,
		twoFactorSchema,
		limen.WithSchemaIDField(schema),
		limen.WithSchemaField(string(TwoFactorSchemaUserIDField), schema.GetIDColumnType()),
		limen.WithSchemaField(string(TwoFactorSchemaSecretField), limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(string(TwoFactorSchemaBackupCodesField), limen.ColumnTypeText, limen.WithNullable(true)),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_two_factors_users_user_id",
			Column:           TwoFactorSchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionRestrict,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithSchemaUniqueIndex("idx_two_factors_user_id", []limen.SchemaField{TwoFactorSchemaUserIDField}),
	)

	return []limen.SchemaIntrospector{
		userWithTwoFactorExtension,
		twoFactorTable,
	}
}

func (t *twoFactorPlugin) FindTwoFactorByUserID(ctx context.Context, userID any) (*TwoFactor, error) {
	twoFactor, err := t.core.FindOne(ctx, t.twoFactorSchema, []limen.Where{
		limen.Eq(t.twoFactorSchema.GetUserIDField(), userID),
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
	return t.core.Delete(ctx, t.twoFactorSchema, []limen.Where{
		limen.Eq(t.twoFactorSchema.GetUserIDField(), userID),
	})
}

func (t *twoFactorPlugin) TOTP() *totp {
	return t.totp
}

func (t *twoFactorPlugin) OTP() *otp {
	return t.otp
}

func (t *twoFactorPlugin) BackupCodes() *backupCodes {
	return t.backupCodes
}

func (t *twoFactorPlugin) encrypt(value string) (string, error) {
	return limen.EncryptXChaCha(value, t.config.secret, nil)
}

func (t *twoFactorPlugin) decrypt(secret string) (string, error) {
	return limen.DecryptXChaCha(secret, t.config.secret, nil)
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

func (t *twoFactorPlugin) setChallengeCookie(ctx *limen.HookContext, token string) {
	t.core.Cookies().SetOnHookCtx(ctx, t.config.cookieName, token, int(t.config.cookieExpiration.Seconds()))
}

func (t *twoFactorPlugin) clearChallengeCookie(w http.ResponseWriter) {
	t.core.Cookies().Delete(w, t.config.cookieName)
}

func (t *twoFactorPlugin) getChallengeFromCookie(r *http.Request) (string, error) {
	val, err := t.core.Cookies().Get(r, t.config.cookieName)
	if err != nil {
		return "", ErrChallengeMissing
	}
	return val, nil
}
