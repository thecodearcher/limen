// Package aegis provides a framework for building authentication systems.
package aegis

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type Aegis struct {
	EmailPassword EmailPasswordFeature
	config        *Config
	core          *AegisCore
}

type AegisCore struct {
	DB             DatabaseAdapter
	DBAction       *DatabaseActionHelper
	Schema         SchemaConfig
	SessionManager *SessionManager
}

type AegisHTTPCore struct {
	Responder              *Responder
	core                   *AegisCore
	authInstance           *Aegis
	config                 *httpConfig
	trustedOriginsPatterns []*regexp.Regexp
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

	sessionManager := newSessionManager(core, config.Session, config.HTTP.cookieConfig)
	core.DBAction = newCommonDatabaseActionsHelper(core)
	core.SessionManager = sessionManager
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

func (a *Aegis) Handler() http.Handler {
	config := a.config.HTTP

	config.basePath = httpx.NormalizePath(config.basePath)

	httpCore := &AegisHTTPCore{
		Responder:              NewResponder(config),
		authInstance:           a,
		config:                 config,
		core:                   a.core,
		trustedOriginsPatterns: a.compileTrustedOrigins(config),
	}

	globalMiddlewares := a.prepareGlobalMiddlewares(config, httpCore)

	router := httpx.NewRouter(globalMiddlewares...)
	a.registerBaseRoutes(router, httpCore, config.basePath)
	a.registerPluginRoutes(router, httpCore, config)

	return router
}

func (a *Aegis) GetSession(req *http.Request) (*AegisSession, error) {
	sessionValidateResult, err := a.core.SessionManager.ValidateSession(req.Context(), req)
	if err != nil {
		return nil, err
	}
	return &AegisSession{
		User:    sessionValidateResult.User,
		Session: sessionValidateResult.Session,
	}, nil
}

func (a *Aegis) registerPluginRoutes(router *httpx.Router, httpCore *AegisHTTPCore, config *httpConfig) {
	for _, feature := range a.config.Features {
		featureConfig := feature.PluginHTTPConfig()
		basePath := featureConfig.BasePath
		override := config.overrides[string(feature.Name())]
		normalizedBasePath := a.normalizePluginPath(config.basePath, basePath, override)
		routeBuilder := &RouteBuilder{
			group: router.Group(normalizedBasePath, featureConfig.Middleware...),
			core:  httpCore,
		}
		feature.RegisterRoutes(httpCore, routeBuilder)
	}
}

func (a *Aegis) prepareGlobalMiddlewares(config *httpConfig, httpCore *AegisHTTPCore) []httpx.Middleware {
	globalMiddlewares := []httpx.Middleware{}
	if config.originCheck {
		globalMiddlewares = append(globalMiddlewares, httpCore.middlewareCheckOrigin())
	}
	if config.csrfProtection {
		globalMiddlewares = append(globalMiddlewares, httpCore.middlewareCSRFProtection())
	}

	rateLimiterRules := a.prepareRateLimiterRules(config.basePath, config)
	rateLimiter := newRateLimiter(config.rateLimiter, httpCore, rateLimiterRules)

	globalMiddlewares = append(globalMiddlewares, rateLimiter.handle)
	globalMiddlewares = append(globalMiddlewares, config.middleware...)
	return globalMiddlewares
}

func (a *Aegis) normalizePluginPath(basePath string, pluginBasePath string, override *PluginHTTPOverride) string {
	if override != nil && override.BasePath != "" {
		pluginBasePath = override.BasePath
	}

	return path.Join(basePath, httpx.NormalizePath(pluginBasePath))
}

func (a *Aegis) prepareRateLimiterRules(basePath string, config *httpConfig) map[string]*RateLimitRule {
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
	config *httpConfig,
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

func (a *Aegis) registerBaseRoutes(router *httpx.Router, httpCore *AegisHTTPCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group: router.Group(basePath),
		core:  httpCore,
	}
	api := NewAegisAPI(httpCore, a.core)
	api.RegisterRoutes(routeBuilder)
}

func (a *Aegis) compileTrustedOrigins(httpConfig *httpConfig) []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, 0, len(httpConfig.trustedOrigins))
	for _, pattern := range httpConfig.trustedOrigins {
		normalizedPattern := pattern
		if !strings.Contains(pattern, "://") {
			normalizedPattern = "*://" + pattern
		}
		regexPattern := globToRegex(normalizedPattern)
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			log.Panicf("failed to compile pattern for trusted origin %s: %v", pattern, err)
		}
		patterns = append(patterns, re)
	}
	return patterns
}
