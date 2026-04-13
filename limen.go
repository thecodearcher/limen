// Package limen is the main package for the Limen authentication library.
package limen

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"path"
)

type Limen struct {
	config *Config
	core   *LimenCore
}

func New(config *Config) (*Limen, error) {
	if config == nil {
		return nil, fmt.Errorf("missing configuration")
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if config.Plugins == nil {
		config.Plugins = []Plugin{}
	}

	limen := &Limen{
		config: config,
	}

	core := &LimenCore{
		config:      config,
		baseURL:     config.BaseURL,
		fullBaseURL: joinURL(config.BaseURL, config.HTTP.basePath),
		db:          config.Database,
		cacheStore:  config.CacheStore,
		Schema:      config.Schema,
		plugins:     make(map[PluginName]Plugin),
		secret:      config.Secret,
	}

	core.cookies = newCookieManager(config.HTTP.cookieConfig, config.Secret)
	sessionManager := newOpaqueSessionManager(core, config.Session)
	core.DBAction = newCommonDatabaseActionsHelper(core)
	core.SessionManager = sessionManager

	discoveredSchemas, err := discoverSchemas(config.Schema, config.Plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to discover schemas: %w", err)
	}
	core.schemaResolver = newFieldResolver(discoveredSchemas)

	// Serialize schemas for CLI if enabled
	if config.CLI != nil && config.CLI.Enabled {
		if err := config.prepareCLIConfig(discoveredSchemas); err != nil {
			return nil, fmt.Errorf("failed to prepare CLI config: %w", err)
		}
	}

	if err := core.initializeSchemas(discoveredSchemas); err != nil {
		return nil, fmt.Errorf("failed to initialize core schemas: %w", err)
	}

	// Initialize plugins
	var smProvider PluginName
	for _, plugin := range config.Plugins {
		if err := plugin.Initialize(core); err != nil {
			return nil, fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
		}
		core.plugins[plugin.Name()] = plugin

		if sp, ok := plugin.(SessionManagerProvider); ok {
			if smProvider != "" {
				return nil, fmt.Errorf("multiple session manager plugins: %s and %s", smProvider, plugin.Name())
			}
			core.SessionManager = sp.SessionManager()
			smProvider = plugin.Name()
		}
	}

	limen.core = core

	return limen, nil
}

func (a *Limen) Handler() http.Handler {
	config := a.config.HTTP

	config.basePath = normalizePath(config.basePath)

	httpCore := &LimenHTTPCore{
		Responder:              newResponder(config, a.core.cookies, a.config.Session.BearerEnabled),
		authInstance:           a,
		config:                 config,
		core:                   a.core,
		trustedOriginsPatterns: compileTrustedOrigins(config.trustedOrigins...),
	}

	globalMiddlewares := prepareGlobalMiddlewares(config, httpCore, a.config.Plugins)

	router := newRouter(httpCore.Responder, globalMiddlewares...)
	if config.hooks != nil {
		router.AddHooks(config.hooks)
	}

	registerBaseRoutes(router, httpCore, a.core, config.basePath)
	registerPluginRoutes(router, a.config.Plugins, httpCore, config)

	return router
}

func (a *Limen) GetSession(req *http.Request) (*ValidatedSession, error) {
	return a.core.SessionManager.ValidateSession(req.Context(), req)
}

// RevokeSession revokes a single session identified by its token.
func (a *Limen) RevokeSession(ctx context.Context, token string) error {
	return a.core.SessionManager.RevokeSession(ctx, token)
}

// RevokeAllSessions revokes every session belonging to the given user.
func (a *Limen) RevokeAllSessions(ctx context.Context, userID any) error {
	return a.core.SessionManager.RevokeAllSessions(ctx, userID)
}

// ListSessions returns all active sessions for the given user.
func (a *Limen) ListSessions(ctx context.Context, userID any) ([]Session, error) {
	return a.core.SessionManager.ListSessions(ctx, userID)
}

// RequestEmailVerification creates a verification token for the user and
// optionally sends the verification email. Returns ErrEmailAlreadyVerified
// when the address is already confirmed.
func (a *Limen) RequestEmailVerification(ctx context.Context, user *User, shouldSendEmail bool) (*Verification, error) {
	return a.core.RequestEmailVerification(ctx, user, shouldSendEmail)
}

// VerifyEmail validates the token, marks the email as verified, and deletes
// the consumed token.
func (a *Limen) VerifyEmail(ctx context.Context, token string) error {
	return a.core.VerifyEmail(ctx, token)
}

// Use retrieves a registered plugin by name and returns it as type T.
// It panics if the plugin is not registered or does not implement T.
//
// T should be gotten from the plugin's API interface.
// For example, if you want to use the credential-password plugin, you can get the API interface like this:
//
//	credentialpasswordAPI := credentialpassword.Use(limen)
//	credentialpasswordAPI.SignInWithCredentialAndPassword(ctx, "user@example.com", "password")
func Use[T any](a *Limen, name PluginName) T {
	plugin, ok := a.core.GetPlugin(name)
	if !ok {
		panic(fmt.Sprintf("limen: plugin %q not registered; add it to Config.Plugins", name))
	}
	typed, ok := plugin.(T)
	if !ok {
		panic(fmt.Sprintf("limen: plugin %q does not implement the requested interface", name))
	}
	return typed
}

// TryUse retrieves a registered plugin by name and returns it as type T.
// Returns the zero value of T and false if the plugin is not registered or
// does not implement T.
//
// Use this when you want to handle missing plugins gracefully instead of panicking.
// If you want to ensure that the plugin is registered, use the Use() function instead.
//
// For example, if you want to use the credential-password plugin, you can get the API interface like this:
//
//	credentialpasswordAPI, ok := limen.TryUse[credentialpassword.API](limen, limen.PluginCredentialPassword)
//	if !ok {
//		return nil, fmt.Errorf("credential password plugin is not registered")
//	}
//	credentialpasswordAPI.SignInWithCredentialAndPassword(ctx, "user@example.com", "password")
func TryUse[T any](a *Limen, name PluginName) (T, bool) {
	plugin, ok := a.core.GetPlugin(name)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := plugin.(T)
	return typed, ok
}

func registerPluginRoutes(router *router, plugins []Plugin, httpCore *LimenHTTPCore, config *httpConfig) {
	for _, plugin := range plugins {
		pluginConfig := plugin.PluginHTTPConfig()
		basePath := pluginConfig.BasePath
		override := config.overrides[string(plugin.Name())]
		normalizedBasePath := normalizePluginPath(config.basePath, basePath, override)
		routeBuilder := &RouteBuilder{
			group: router.Group(normalizedBasePath, pluginConfig.Middleware...),
			core:  httpCore,
		}

		plugin.RegisterRoutes(httpCore, routeBuilder)

		if pluginConfig.Hooks != nil {
			router.AddHooks(pluginConfig.Hooks)
		}
	}
}

func prepareGlobalMiddlewares(config *httpConfig, httpCore *LimenHTTPCore, plugins []Plugin) []Middleware {
	globalMiddlewares := []Middleware{middlewareAdditionalFieldsContext()}

	if config.originCheck {
		globalMiddlewares = append(globalMiddlewares, httpCore.middlewareCheckOrigin())
	}
	if config.csrfProtection {
		globalMiddlewares = append(globalMiddlewares, httpCore.middlewareCSRFProtection())
	}

	rateLimiterRules := prepareRateLimiterRules(config.basePath, config, plugins)
	rateLimiter := newRateLimiter(config.rateLimiter, httpCore, rateLimiterRules)
	globalMiddlewares = append(globalMiddlewares, rateLimiter.handle)

	globalMiddlewares = append(globalMiddlewares, config.middleware...)
	return globalMiddlewares
}

func prepareRateLimiterRules(basePath string, httpConfig *httpConfig, plugins []Plugin) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)

	customRules := httpConfig.rateLimiter.customRules

	for _, plugin := range plugins {
		pluginRules := processPluginRateLimitRules(plugin, basePath, httpConfig, customRules)
		maps.Copy(rules, pluginRules)
	}

	resolvedCustomRules := processCustomRateLimitRules(basePath, customRules)
	maps.Copy(rules, resolvedCustomRules)

	return rules
}

func processPluginRateLimitRules(plugin Plugin, basePath string, httpConfig *httpConfig, customRules map[string]*RateLimitRule) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)
	pluginConfig := plugin.PluginHTTPConfig()
	override := httpConfig.overrides[string(plugin.Name())]
	normalizedBasePath := normalizePluginPath(basePath, pluginConfig.BasePath, override)

	if len(pluginConfig.RateLimitRules) == 0 {
		return rules
	}

	for _, rule := range pluginConfig.RateLimitRules {
		finalRule := resolveRuleOverride(rule, customRules)
		completePath := path.Join(normalizedBasePath, rule.path)

		if err := compileAndSetRulePattern(finalRule, completePath); err != nil {
			log.Panicf("failed to compile pattern for path %s: %v", completePath, err)
		}

		rules[completePath] = finalRule
	}

	return rules
}
