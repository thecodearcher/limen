// Package aegis provides a framework for building authentication systems.
package aegis

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"path"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type Aegis struct {
	EmailPassword  EmailPasswordFeature
	config         *Config
	sessionManager *SessionManager
	core           *AegisCore
}

type AegisCore struct {
	DB             DatabaseAdapter
	DBAction       *DatabaseActionHelper
	Schema         SchemaConfig
	SessionManager *SessionManager
	Responder      *Responder
}

type AegisHTTPCore struct {
	Responder    *Responder
	AuthInstance *Aegis
	Config       *HTTPConfig
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
		DB:     config.Database,
		Schema: config.Schema,
	}

	sessionManager := newSessionManager(core, config.Session)
	core.DBAction = newCommonDatabaseActionsHelper(core)
	core.SessionManager = sessionManager
	aegis.sessionManager = sessionManager
	aegis.core = core

	for _, feature := range config.Features {
		if err := feature.Initialize(core); err != nil {
			return nil, fmt.Errorf("failed to initialize feature %s: %w", feature.Name(), err)
		}

		switch feature.Name() {
		case FeatureEmailPassword:
			aegis.EmailPassword = feature.(EmailPasswordFeature)
		}
	}

	return aegis, nil
}

func (a *Aegis) Handler(opts ...HTTPConfigOption) http.Handler {
	config := NewDefaultHTTPConfig(opts...)

	if config.rateLimiter != nil {
		config.rateLimiter.validate()
	}

	config.basePath = httpx.NormalizePath(config.basePath)

	httpCore := &AegisHTTPCore{
		Responder:    NewResponder(config),
		AuthInstance: a,
		Config:       config,
	}

	rateLimiterRules := a.prepareRateLimiterRules(config.basePath, config)

	rateLimiter := NewRateLimiter(config.rateLimiter, httpCore, rateLimiterRules)
	globalMiddlewares := append([]httpx.Middleware{rateLimiter.Handle}, config.middleware...)

	router := httpx.NewRouter(globalMiddlewares...)
	registerBaseRoutes(router, httpCore, a.core, config.basePath)

	for _, feature := range a.config.Features {
		featureConfig := feature.PluginHTTPConfig()
		basePath := featureConfig.BasePath
		override := config.overrides[string(feature.Name())]
		normalizedBasePath := a.normalizePluginPath(config.basePath, basePath, override)
		routeBuilder := &RouteBuilder{
			group:         router.Group(normalizedBasePath, featureConfig.Middleware...),
			AegisHTTPCore: httpCore,
		}
		feature.RegisterRoutes(routeBuilder)
	}

	return router
}

func (a *Aegis) GetSession(req *http.Request) (*AegisSession, error) {
	sessionValidateResult, err := a.sessionManager.ValidateSession(req.Context(), req)
	if err != nil {
		return nil, err
	}
	return &AegisSession{
		User:    sessionValidateResult.User,
		Session: sessionValidateResult.Session,
	}, nil
}

func (a *Aegis) normalizePluginPath(basePath string, pluginBasePath string, override *PluginHTTPOverride) string {
	if override != nil && override.BasePath != "" {
		pluginBasePath = override.BasePath
	}

	return path.Join(basePath, httpx.NormalizePath(pluginBasePath))
}

func (a *Aegis) prepareRateLimiterRules(basePath string, config *HTTPConfig) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)

	customRules := config.rateLimiter.customRules

	// Process feature rules
	for _, feature := range a.config.Features {
		featureRules := a.processFeatureRateLimitRules(feature, basePath, config, customRules)
		maps.Copy(rules, featureRules)
	}

	resolvedCustomRules := a.processCustomRateLimitRules(basePath, customRules)
	maps.Copy(rules, resolvedCustomRules)

	return rules
}

func (a *Aegis) processFeatureRateLimitRules(
	feature Feature,
	basePath string,
	config *HTTPConfig,
	customRules map[string]*RateLimitRule,
) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)
	featureConfig := feature.PluginHTTPConfig()
	override := config.overrides[string(feature.Name())]
	normalizedBasePath := a.normalizePluginPath(basePath, featureConfig.BasePath, override)

	if len(featureConfig.RateLimitRules) == 0 {
		return rules
	}

	for _, rule := range featureConfig.RateLimitRules {
		finalRule := a.resolveRuleOverride(rule, customRules)
		completePath := path.Join(normalizedBasePath, rule.path)

		if err := a.compileAndSetRulePattern(finalRule, completePath); err != nil {
			log.Panicf("failed to compile pattern for path %s: %v", completePath, err)
		}

		rules[completePath] = finalRule
	}

	return rules
}

func (a *Aegis) processCustomRateLimitRules(basePath string, customRules map[string]*RateLimitRule) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)

	for pattern, rule := range customRules {
		completePath := path.Join(basePath, pattern)

		if err := a.compileAndSetRulePattern(rule, completePath); err != nil {
			log.Panicf("failed to compile pattern for path %s: %v", completePath, err)
		}

		rules[completePath] = rule
	}

	return rules
}

// compileAndSetRulePattern compiles the pattern and sets it on the rule
func (a *Aegis) compileAndSetRulePattern(rule *RateLimitRule, completePath string) error {
	compiledPattern, err := compilePattern(completePath)
	if err != nil {
		return fmt.Errorf("failed to compile pattern: %w", err)
	}

	rule.path = completePath
	rule.pathRegex = compiledPattern
	return nil
}

func (a *Aegis) resolveRuleOverride(rule *RateLimitRule, customRules map[string]*RateLimitRule) *RateLimitRule {
	if customRule, exists := customRules[rule.path]; exists {
		delete(customRules, rule.path)
		return customRule
	}
	return rule
}

func registerBaseRoutes(router *httpx.Router, httpCore *AegisHTTPCore, core *AegisCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group:         router.Group(basePath),
		AegisHTTPCore: httpCore,
	}
	api := NewAegisAPI(httpCore, core)
	api.RegisterRoutes(routeBuilder)
}
