// Package passkey provides WebAuthn passkey authentication for Limen.
package passkey

import (
	"fmt"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

type passkeyPlugin struct {
	core          *limen.LimenCore
	config        *config
	passkeySchema *passkeySchema
	webAuthn      *webauthn.WebAuthn
}

// New creates a passkey plugin.
func New(opts ...ConfigOption) *passkeyPlugin {
	cfg := &config{
		rpName: "limen-auth",
		authenticatorSelection: AuthenticatorSelection{
			UserVerification: UserVerificationRequired,
			ResidentKey:      ResidentKeyPreferred,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &passkeyPlugin{config: cfg}
}

func (p *passkeyPlugin) Name() limen.PluginName {
	return limen.PluginPasskey
}

func (p *passkeyPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core
	p.resolveRelyingParty(core.GetBaseURL())

	webAuthn, err := webauthn.New(&webauthn.Config{
		RPID:                   p.config.rpID,
		RPDisplayName:          p.config.rpName,
		RPOrigins:              p.config.allowedOrigins,
		AttestationPreference:  protocol.PreferNoAttestation,
		AuthenticatorSelection: p.config.authenticatorSelection,
	})
	if err != nil {
		return fmt.Errorf("passkey: %w", err)
	}

	p.webAuthn = webAuthn

	return nil
}

func (p *passkeyPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/passkey",
	}
}

func (p *passkeyPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
}

func (p *passkeyPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	p.passkeySchema = newPasskeySchema()

	return []limen.SchemaIntrospector{buildPasskeyTableDef(schema, p.passkeySchema)}
}

func (p *passkeyPlugin) resolveRelyingParty(baseURL string) {
	if p.config.rpID == "" {
		p.config.rpID = parseHost(baseURL)
	}
	if len(p.config.allowedOrigins) == 0 {
		p.config.allowedOrigins = []string{baseURL}
	}
}

func parseHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}
