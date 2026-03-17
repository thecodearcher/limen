package sessionjwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/thecodearcher/limen"
)

type sessionJWTPlugin struct {
	core               *limen.LimenCore
	config             *config
	refreshTokenSchema *refreshTokenSchema
	blacklistSchema    *blacklistSchema
	blacklist          blacklistStore
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
		subjectEncoder:       func(user *limen.User) string { return fmt.Sprintf("%v", user.ID) },
		subjectResolver:      func(subject string) (any, error) { return subject, nil },
		refreshUser:          false,
		blacklistStoreType:   limen.StoreTypeCache,
	}

	for _, opt := range opts {
		opt(cfg)
	}
	return &sessionJWTPlugin{config: cfg}
}

func (p *sessionJWTPlugin) Name() limen.PluginName {
	return limen.PluginSessionJWT
}

func (p *sessionJWTPlugin) Initialize(core *limen.LimenCore) error {
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

	if err := p.config.resolveKeys(core.Secret()); err != nil {
		return err
	}

	if p.config.blacklistEnabled {
		p.blacklist = p.determineBlacklistStore()
	}

	return nil
}

func (p *sessionJWTPlugin) determineBlacklistStore() blacklistStore {
	if p.config.blacklistStoreType == limen.StoreTypeDatabase {
		return &dbBlacklistStore{core: p.core, schema: p.blacklistSchema}
	}
	return &cacheBlacklistStore{
		cache:  p.core.CacheStore(),
		prefix: p.core.CacheKeyPrefix(),
	}
}

func (p *sessionJWTPlugin) SessionManager() limen.SessionManager {
	return &jwtSessionManager{plugin: p}
}

func (p *sessionJWTPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath:   "/",
		Middleware: []limen.Middleware{},
	}
}

func (p *sessionJWTPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	if !p.config.refreshTokenEnabled {
		return
	}
	handlers := newJWTHandlers(p, httpCore)
	routeBuilder.POST("/refresh", "session-jwt-refresh", handlers.Refresh)
}

func (p *sessionJWTPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	var schemas []limen.SchemaIntrospector

	if p.config.refreshTokenEnabled {
		p.refreshTokenSchema = newRefreshTokenSchema()
		schemas = append(schemas, buildRefreshTokenTableDef(schema, p.refreshTokenSchema))
	}

	if p.config.blacklistEnabled && p.config.blacklistStoreType != limen.StoreTypeCache {
		p.blacklistSchema = newBlacklistSchema()
		schemas = append(schemas, buildBlacklistTableDef(p.blacklistSchema))
	}

	return schemas
}
