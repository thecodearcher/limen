// Package passkey provides WebAuthn / passkey authentication for Limen.
//
// The plugin implements the four ceremonies required for a complete
// passkey experience plus credential management:
//
//   GET  /passkey/generate-register-options   - kick off a registration
//   POST /passkey/verify-registration         - finish registration
//   GET  /passkey/generate-authenticate-options - kick off an authentication
//   POST /passkey/verify-authentication       - finish authentication, sets session
//   GET  /passkey/list                        - list the current user's passkeys
//   POST /passkey/delete                      - delete a passkey by id
//   POST /passkey/update                      - rename a passkey
//
// It is modelled after better-auth's passkey plugin and uses
// github.com/go-webauthn/webauthn for the protocol-level work.
package passkey

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

type passkeyPlugin struct {
	core         *limen.LimenCore
	config       *config
	userSchema   *limen.UserSchema
	passkeySchema *passkeySchema
	verifSchema  *limen.VerificationSchema
	dbAction     *limen.DatabaseActionHelper
	webauthn     *webauthn.WebAuthn
}

// New returns a passkey plugin configured with sensible defaults. At
// least one of RPID or origins should be set in production; the
// defaults are tuned for "localhost" development.
func New(opts ...ConfigOption) *passkeyPlugin {
	cfg := &config{
		rpName:              defaultRPName,
		challengeExpiration: defaultChallengeExpiration,
		challengeCookieName: defaultChallengeCookieName,
		requireSessionToReg: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &passkeyPlugin{config: cfg}
}

func (p *passkeyPlugin) Name() limen.PluginName { return limen.PluginPasskey }

func (p *passkeyPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core
	p.userSchema = core.Schema.User
	p.verifSchema = core.Schema.Verification
	p.dbAction = core.DBAction

	if p.config == nil {
		return fmt.Errorf("passkey: config is required")
	}
	if p.config.challengeExpiration <= 0 {
		return fmt.Errorf("passkey: challenge expiration must be positive")
	}

	// Derive RP origins + RP ID from Limen's BaseURL when unset.
	baseURL := core.GetBaseURL()
	if len(p.config.origins) == 0 && baseURL != "" {
		p.config.origins = []string{baseURL}
	}
	if p.config.rpID == "" {
		p.config.rpID = inferRPID(baseURL)
	}
	if len(p.config.origins) == 0 {
		return ErrOriginRequired
	}

	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName:         p.config.rpName,
		RPID:                  p.config.rpID,
		RPOrigins:             p.config.origins,
		AuthenticatorSelection: authSelectionOrDefault(p.config.authenticatorSelect),
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    p.config.challengeExpiration,
				TimeoutUVD: p.config.challengeExpiration,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    p.config.challengeExpiration,
				TimeoutUVD: p.config.challengeExpiration,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("passkey: webauthn init: %w", err)
	}
	p.webauthn = w

	return nil
}

func (p *passkeyPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/passkey",
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("/generate-register-options", 10, time.Minute),
			limen.NewRateLimitRule("/verify-registration", 10, time.Minute),
			limen.NewRateLimitRule("/generate-authenticate-options", 20, time.Minute),
			limen.NewRateLimitRule("/verify-authentication", 20, time.Minute),
		},
	}
}

func (p *passkeyPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	h := newPasskeyHandlers(p, httpCore)
	routeBuilder.ProtectedGET("/generate-register-options", "passkey-gen-register", h.GenerateRegistrationOptions)
	routeBuilder.ProtectedPOST("/verify-registration", "passkey-verify-register", h.VerifyRegistration)
	// Authentication endpoints do NOT require a session — they create one.
	routeBuilder.GET("/generate-authenticate-options", "passkey-gen-auth", h.GenerateAuthenticationOptions)
	routeBuilder.POST("/verify-authentication", "passkey-verify-auth", h.VerifyAuthentication)
	routeBuilder.ProtectedGET("/list", "passkey-list", h.ListPasskeys)
	routeBuilder.ProtectedPOST("/delete", "passkey-delete", h.DeletePasskey)
	routeBuilder.ProtectedPOST("/update", "passkey-update", h.UpdatePasskey)
}

func (p *passkeyPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	s := newPasskeySchema()
	p.passkeySchema = s

	table := limen.NewSchemaDefinitionForTable(
		limen.SchemaName(PasskeySchemaTableName),
		PasskeySchemaTableName,
		s,
		limen.WithSchemaIDField(schema),
		limen.WithSchemaField(string(PasskeySchemaUserIDField), schema.GetIDColumnType()),
		limen.WithSchemaField(string(PasskeySchemaNameField), limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(string(PasskeySchemaCredentialIDField), limen.ColumnTypeText),
		limen.WithSchemaField(string(PasskeySchemaPublicKeyField), limen.ColumnTypeText),
		limen.WithSchemaField(string(PasskeySchemaCounterField), limen.ColumnTypeInt64, limen.WithDefaultValue("0")),
		limen.WithSchemaField(string(PasskeySchemaDeviceTypeField), limen.ColumnTypeString),
		limen.WithSchemaField(string(PasskeySchemaBackedUpField), limen.ColumnTypeBool, limen.WithDefaultValue("false")),
		limen.WithSchemaField(string(PasskeySchemaTransportsField), limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(string(PasskeySchemaAAGUIDField), limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(string(limen.SchemaCreatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaField(string(limen.SchemaUpdatedAtField), limen.ColumnTypeTime),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_passkeys_users_user_id",
			Column:           PasskeySchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithSchemaUniqueIndex("idx_passkeys_credential_id", []limen.SchemaField{PasskeySchemaCredentialIDField}),
	)

	return []limen.SchemaIntrospector{table}
}

// inferRPID derives the WebAuthn Relying Party ID from a base URL.
// Defaults to "localhost" if the URL can't be parsed or has no host.
func inferRPID(baseURL string) string {
	if baseURL == "" {
		return "localhost"
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return "localhost"
	}
	host := u.Hostname()
	if host == "" {
		return "localhost"
	}
	return host
}
