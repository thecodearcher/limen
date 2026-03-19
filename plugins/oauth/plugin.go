package oauth

import (
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type oauthPlugin struct {
	core          *limen.LimenCore
	accountSchema *limen.AccountSchema
	config        *config
	providers     map[string]Provider
	stateStore    StateStore
	httpCore      *limen.LimenHTTPCore
	cookies       *limen.CookieManager
}

func New(opts ...ConfigOption) *oauthPlugin {
	cfg := &config{
		cookieName: "limen_oauth",
		cookieTTL:  10 * time.Minute,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &oauthPlugin{
		config: cfg,
	}
}

func (o *oauthPlugin) Name() limen.PluginName {
	return limen.PluginOAuth
}

func (o *oauthPlugin) Initialize(core *limen.LimenCore) error {
	o.core = core
	o.cookies = core.Cookies()
	o.accountSchema = core.Schema.Account
	if len(o.config.secret) == 0 {
		if base := core.Secret(); len(base) == 32 {
			o.config.secret = base
		}
	}

	if len(o.config.secret) != 32 {
		return fmt.Errorf("oauth: secret must be 32 bytes, got %d (set oauth.WithSecret or Config.Secret)", len(o.config.secret))
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

func (o *oauthPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/oauth",
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("/:provider/authorize", 10, time.Minute),
			limen.NewRateLimitRule("/:provider/callback", 10, time.Minute),
			limen.NewRateLimitRule("/:provider/link", 10, time.Minute),
			limen.NewRateLimitRule("/:provider/unlink", 10, time.Minute),
			limen.NewRateLimitRule("/:provider/token", 20, time.Minute),
			limen.NewRateLimitRule("/:provider/token/refresh", 10, time.Minute),
		},
	}
}

func (o *oauthPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newOAuthHandlers(o, httpCore)
	o.httpCore = httpCore
	routeBuilder.GET("/:provider/authorize", "oauth-authorize", handlers.SignInWithOAuth)
	routeBuilder.GET("/:provider/callback", "oauth-callback", handlers.Callback)
	routeBuilder.ProtectedGET("/:provider/link", "oauth-link-authorize", handlers.LinkAccountWithOAuth)
	routeBuilder.ProtectedGET("/accounts", "oauth-list-accounts", handlers.ListAccounts)
	routeBuilder.ProtectedDELETE("/:provider/unlink", "oauth-unlink-account", handlers.UnlinkAccount)
	routeBuilder.ProtectedGET("/:provider/tokens", "oauth-get-tokens", handlers.GetTokens)
	routeBuilder.ProtectedPOST("/:provider/tokens/refresh", "oauth-refresh-tokens", handlers.RefreshAccessToken)
}

func (o *oauthPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	return []limen.SchemaIntrospector{}
}
