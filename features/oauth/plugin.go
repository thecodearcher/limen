package oauth

import (
	"fmt"
	"time"

	"github.com/thecodearcher/aegis"
)

type oauthFeature struct {
	core          *aegis.AegisCore
	accountSchema *aegis.AccountSchema
	config        *config
	providers     map[string]Provider
	stateStore    StateStore
	httpCore      *aegis.AegisHTTPCore
	cookies       *aegis.CookieManager
}

func New(opts ...ConfigOption) *oauthFeature {
	cfg := &config{
		cookieName: "aegis_oauth",
		cookieTTL:  10 * time.Minute,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &oauthFeature{
		config: cfg,
	}
}

func (o *oauthFeature) Name() aegis.FeatureName {
	return aegis.FeatureOAuth
}

func (o *oauthFeature) Initialize(core *aegis.AegisCore) error {
	o.core = core
	o.cookies = core.Cookies()
	o.accountSchema = core.Schema.Account
	if len(o.config.secret) != 32 {
		return fmt.Errorf("oauth: secret must be 32 bytes, got %d", len(o.config.secret))
	}

	if len(o.config.providers) == 0 {
		return fmt.Errorf("oauth: at least one provider must be registered via oauth.WithProvider()")
	}

	o.providers = o.config.providers

	ttl := o.config.cookieTTL
	if o.config.useDatabaseState {
		o.stateStore = newDatabaseStateStore(core, ttl)
	} else {
		o.stateStore = newStatelessStateStore(o.config.secret, ttl)
	}

	return nil
}

func (o *oauthFeature) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		BasePath: "/oauth",
		RateLimitRules: []*aegis.RateLimitRule{
			aegis.NewRateLimitRule("/:provider/authorize", 10, time.Minute),
			aegis.NewRateLimitRule("/:provider/callback", 10, time.Minute),
			aegis.NewRateLimitRule("/:provider/link", 10, time.Minute),
			aegis.NewRateLimitRule("/:provider/unlink", 10, time.Minute),
		},
	}
}

func (o *oauthFeature) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	handlers := newOAuthHandlers(o, httpCore)
	o.httpCore = httpCore
	routeBuilder.GET("/:provider/authorize", "oauth-authorize", handlers.SignInWithOAuth)
	routeBuilder.GET("/:provider/callback", "oauth-callback", handlers.Callback)
	routeBuilder.ProtectedGET("/:provider/link", "oauth-link-authorize", handlers.LinkAccountWithOAuth)
	routeBuilder.ProtectedGET("/accounts", "oauth-list-accounts", handlers.ListAccounts)
	routeBuilder.ProtectedDELETE("/:provider/unlink", "oauth-unlink-account", handlers.UnlinkAccount)
}

func (o *oauthFeature) GetSchemas(schema *aegis.SchemaConfig) []aegis.SchemaIntrospector {
	return []aegis.SchemaIntrospector{}
}
