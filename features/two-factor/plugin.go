package twofactor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/thecodearcher/aegis"
)

type twoFactorFeature struct {
	core            *aegis.AegisCore
	twoFactorSchema *twoFactorSchema
	userSchema      *userWithTwoFactorSchema
	config          *config
	totp            *totp
	otp             *otp
	backupCodes     *backupCodes
}

func (t *twoFactorFeature) Name() aegis.FeatureName {
	return aegis.FeatureTwoFactor
}

func New(opts ...ConfigOption) *twoFactorFeature {
	config := &config{
		secret: getTOTPSecret(),
		totp:   NewDefaultTOTPConfig(),
		otp:    NewDefaultOTPConfig(),
	}
	for _, opt := range opts {
		opt(config)
	}

	return &twoFactorFeature{
		config: config,
	}
}

func (t *twoFactorFeature) Initialize(core *aegis.AegisCore) error {
	t.core = core
	t.totp = newDefaultTOTP(t, t.config.totp)
	t.backupCodes = newBackupCodes(t, t.config.backupCodes)
	if t.config.otp.enabled {
		t.otp = newDefaultOTP(t, t.config.otp)
	}
	return nil
}

func (t *twoFactorFeature) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		BasePath:   "/two-factor",
		Middleware: []aegis.Middleware{},
		Hooks: &aegis.Hooks{
			After: &aegis.Hook{
				PathMatcher: func(ctx *aegis.HookContext) bool {
					return ctx.RouteID == "signin" || ctx.RouteID == "signup"
				},
				Run: func(ctx *aegis.HookContext) bool {
					fmt.Printf("Before request for two-factor %s %s\n", ctx.Request.Method, ctx.Request.URL.Path)
					fmt.Printf("Status code: %v\n", ctx.Response)
					fmt.Printf("Body: %v\n", ctx)
					ctx.Response.Write([]byte("Hello, world!"))
					ctx.Response.WriteHeader(http.StatusBadRequest)
					return false
				},
			},
		},
	}
}

func (t *twoFactorFeature) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	handlers := newTwoFactorHandlers(t, httpCore.Responder)

	// Global endpoints
	routeBuilder.ProtectedPOST("/initiate-setup", "two-factor-initiate-setup", handlers.InitiateTwoFactorSetup)
	routeBuilder.ProtectedPOST("/finalize-setup", "two-factor-finalize-setup", handlers.FinalizeTwoFactorSetup)
	routeBuilder.ProtectedPOST("/disable", "two-factor-disable", handlers.Disable)

	t.totp.registerRoutes(httpCore, routeBuilder)
	t.backupCodes.registerRoutes(httpCore, routeBuilder)

	// Register OTP routes
	if t.config.otp.enabled {
		t.otp.registerRoutes(httpCore, routeBuilder)
	}
}

func (t *twoFactorFeature) GetSchemas(schema *aegis.SchemaConfig) []aegis.SchemaIntrospector {
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

func (t *twoFactorFeature) FindTwoFactorByUserID(ctx context.Context, userID any) (*TwoFactor, error) {
	twoFactor, err := t.core.FindOne(ctx, t.twoFactorSchema, []aegis.Where{
		aegis.Eq(t.twoFactorSchema.GetUserIDField(), userID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return twoFactor.(*TwoFactor), nil
}

func (t *twoFactorFeature) CreateTwoFactor(ctx context.Context, userID any, encryptedSecret string, encryptedBackupCodes string) error {
	twoFactor := &TwoFactor{
		UserID:      userID,
		Secret:      encryptedSecret,
		BackupCodes: encryptedBackupCodes,
	}
	return t.core.Create(ctx, t.twoFactorSchema, twoFactor, nil)
}

func (t *twoFactorFeature) DeleteTwoFactor(ctx context.Context, userID any) error {
	return t.core.Delete(ctx, t.twoFactorSchema, []aegis.Where{
		aegis.Eq(t.twoFactorSchema.GetUserIDField(), userID),
	})
}

func (t *twoFactorFeature) encrypt(secret string) (string, error) {
	return aegis.EncryptXChaCha(secret, t.config.secret, nil)
}

func (t *twoFactorFeature) decrypt(secret string) (string, error) {
	return aegis.DecryptXChaCha(secret, t.config.secret, nil)
}
