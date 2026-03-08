package sessionjwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/thecodearcher/aegis"
)

type sessionJWTPlugin struct {
	core               *aegis.AegisCore
	config             *config
	refreshTokenSchema *refreshTokenSchema
	blacklistSchema    *blacklistSchema
}

// New creates a new session-jwt plugin. When registered, it replaces the
// default opaque session manager with a JWT-based one.
func New(opts ...ConfigOption) *sessionJWTPlugin {
	cfg := &config{
		signingMethod:        jwt.SigningMethodHS256,
		accessTokenDuration:  15 * time.Minute,
		refreshTokenDuration: 7 * 24 * time.Hour,
		refreshTokenRotation: true,
		blacklistEnabled:     false,
		refreshTokenEnabled:  true,
		subjectEncoder:       func(user *aegis.User) string { return fmt.Sprintf("%v", user.ID) },
		subjectResolver:      func(subject string) (any, error) { return subject, nil },
		refreshUser:          false,
	}

	for _, opt := range opts {
		opt(cfg)
	}
	return &sessionJWTPlugin{config: cfg}
}

func (p *sessionJWTPlugin) Name() aegis.PluginName {
	return aegis.PluginSessionJWT
}

func (p *sessionJWTPlugin) Initialize(core *aegis.AegisCore) error {
	p.core = core

	if p.config.accessTokenDuration <= 0 {
		return fmt.Errorf("session-jwt: accessTokenDuration must be positive")
	}
	if p.config.refreshTokenEnabled && p.config.refreshTokenDuration <= 0 {
		return fmt.Errorf("session-jwt: refreshTokenDuration must be positive")
	}
	if p.config.refreshTokenEnabled && p.config.accessTokenDuration >= p.config.refreshTokenDuration {
		return fmt.Errorf("session-jwt: accessTokenDuration (%s) must be less than refreshTokenDuration (%s)",
			p.config.accessTokenDuration, p.config.refreshTokenDuration)
	}

	if p.config.issuer == "" {
		p.config.issuer = core.GetBaseURL()
	}
	if p.config.issuer == "" {
		return fmt.Errorf("session-jwt: issuer is required; use WithIssuer or set Config.BaseURL")
	}

	if len(p.config.audience) == 0 {
		if base := core.GetBaseURL(); base != "" {
			p.config.audience = []string{base}
		}
	}
	if len(p.config.audience) == 0 {
		return fmt.Errorf("session-jwt: audience is required; use WithAudience or set Config.BaseURL")
	}

	if err := p.config.resolveKeys(core.SigningSecret()); err != nil {
		return err
	}

	return nil
}

func (p *sessionJWTPlugin) SessionManager() aegis.SessionManager {
	return &jwtSessionManager{plugin: p}
}

func (p *sessionJWTPlugin) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		BasePath:   "/",
		Middleware: []aegis.Middleware{},
	}
}

func (p *sessionJWTPlugin) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	if !p.config.refreshTokenEnabled {
		return
	}
	handlers := newJWTHandlers(p, httpCore)
	routeBuilder.POST("/refresh", "session-jwt-refresh", handlers.Refresh)
}

func (p *sessionJWTPlugin) GetSchemas(schema *aegis.SchemaConfig) []aegis.SchemaIntrospector {
	var schemas []aegis.SchemaIntrospector

	if p.config.refreshTokenEnabled {
		p.refreshTokenSchema = newRefreshTokenSchema()
		schemas = append(schemas, buildRefreshTokenTableDef(schema, p.refreshTokenSchema))
	}

	if p.config.blacklistEnabled {
		p.blacklistSchema = newBlacklistSchema()
		schemas = append(schemas, buildBlacklistTableDef(p.blacklistSchema))
	}

	return schemas
}
