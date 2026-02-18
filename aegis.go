// Package aegis is the main package for the Aegis authentication library.
package aegis

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"path"
)

type Aegis struct {
	EmailPassword CredentialPasswordFeature
	config        *Config
	core          *AegisCore
}

func New(config *Config) (*Aegis, error) {
	if config == nil {
		return nil, fmt.Errorf("missing configuration")
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if config.Features == nil {
		config.Features = []Feature{}
	}

	aegis := &Aegis{
		config: config,
	}
	core := &AegisCore{
		config:      config,
		baseURL:     config.BaseURL,
		fullBaseURL: joinURL(config.BaseURL, config.HTTP.basePath),
		db:          config.Database,
		Schema:      config.Schema,
		features:    make(map[FeatureName]Feature),
	}

	sessionManager := newOpaqueSessionManager(core, config.Session, config.HTTP.cookieConfig)
	core.DBAction = newCommonDatabaseActionsHelper(core)
	core.SessionManager = sessionManager

	discoveredSchemas, err := discoverSchemas(config.Schema, config.Features)
	if err != nil {
		return nil, fmt.Errorf("failed to discover schemas: %w", err)
	}
	core.schemaResolver = newFieldResolver(discoveredSchemas)

	aegis.core = core

	// Serialize schemas for CLI if enabled
	if config.CLI != nil && config.CLI.Enabled {
		if err := config.prepareCLIConfig(discoveredSchemas); err != nil {
			return nil, fmt.Errorf("failed to prepare CLI config: %w", err)
		}
	}

	if err := core.initializeSchemas(discoveredSchemas); err != nil {
		return nil, fmt.Errorf("failed to initialize core schemas: %w", err)
	}

	// Initialize features
	for _, feature := range config.Features {
		if err := feature.Initialize(core); err != nil {
			return nil, fmt.Errorf("failed to initialize feature %s: %w", feature.Name(), err)
		}

		// Register feature in the plugin registry
		core.features[feature.Name()] = feature

		switch feature.Name() {
		case FeatureCredentialPassword:
			aegis.EmailPassword = feature.(CredentialPasswordFeature)
		}
	}

	return aegis, nil
}

func (a *Aegis) Handler() http.Handler {
	config := a.config.HTTP

	config.basePath = NormalizePath(config.basePath)
	allUrls := []string{a.core.GetBaseURL()}
	allUrls = append(allUrls, config.trustedOrigins...)

	httpCore := &AegisHTTPCore{
		Responder:              newResponder(config),
		authInstance:           a,
		config:                 config,
		core:                   a.core,
		trustedOriginsPatterns: compileTrustedOrigins(allUrls...),
	}

	globalMiddlewares := prepareGlobalMiddlewares(config, httpCore, a.config.Features)

	router := NewRouter(httpCore.Responder, globalMiddlewares...)
	if config.hooks != nil {
		router.AddHooks(config.hooks)
	}

	registerBaseRoutes(router, httpCore, a.core, config.basePath)
	registerPluginRoutes(router, a.config.Features, httpCore, config)

	return router
}

func (a *Aegis) GetSession(req *http.Request) (*ValidatedSession, error) {
	return a.core.SessionManager.ValidateSession(req.Context(), req)
}

func registerPluginRoutes(router *Router, features []Feature, httpCore *AegisHTTPCore, config *httpConfig) {
	for _, feature := range features {
		featureConfig := feature.PluginHTTPConfig()
		basePath := featureConfig.BasePath
		override := config.overrides[string(feature.Name())]
		normalizedBasePath := normalizePluginPath(config.basePath, basePath, override)
		routeBuilder := &RouteBuilder{
			group: router.Group(normalizedBasePath, featureConfig.Middleware...),
			core:  httpCore,
		}

		feature.RegisterRoutes(httpCore, routeBuilder)

		if featureConfig.Hooks != nil {
			router.AddHooks(featureConfig.Hooks)
		}
	}
}

func prepareGlobalMiddlewares(config *httpConfig, httpCore *AegisHTTPCore, features []Feature) []Middleware {
	globalMiddlewares := []Middleware{middlewareAdditionalFieldsContext()}

	if config.originCheck {
		globalMiddlewares = append(globalMiddlewares, httpCore.middlewareCheckOrigin())
	}
	if config.csrfProtection {
		globalMiddlewares = append(globalMiddlewares, httpCore.middlewareCSRFProtection())
	}

	rateLimiterRules := prepareRateLimiterRules(config.basePath, config, features)
	rateLimiter := newRateLimiter(config.rateLimiter, httpCore, rateLimiterRules)
	globalMiddlewares = append(globalMiddlewares, rateLimiter.handle)

	globalMiddlewares = append(globalMiddlewares, config.middleware...)
	return globalMiddlewares
}

func prepareRateLimiterRules(basePath string, httpConfig *httpConfig, features []Feature) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)

	customRules := httpConfig.rateLimiter.customRules

	for _, feature := range features {
		featureRules := processFeatureRateLimitRules(feature, basePath, httpConfig, customRules)
		maps.Copy(rules, featureRules)
	}

	resolvedCustomRules := processCustomRateLimitRules(basePath, customRules)
	maps.Copy(rules, resolvedCustomRules)

	return rules
}

func processFeatureRateLimitRules(feature Feature, basePath string, httpConfig *httpConfig, customRules map[string]*RateLimitRule) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)
	featureConfig := feature.PluginHTTPConfig()
	override := httpConfig.overrides[string(feature.Name())]
	normalizedBasePath := normalizePluginPath(basePath, featureConfig.BasePath, override)

	if len(featureConfig.RateLimitRules) == 0 {
		return rules
	}

	for _, rule := range featureConfig.RateLimitRules {
		finalRule := resolveRuleOverride(rule, customRules)
		completePath := path.Join(normalizedBasePath, rule.path)

		if err := compileAndSetRulePattern(finalRule, completePath); err != nil {
			log.Panicf("failed to compile pattern for path %s: %v", completePath, err)
		}

		rules[completePath] = finalRule
	}

	return rules
}
