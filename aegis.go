// Package aegis provides a framework for building authentication systems.
package aegis

import (
	"fmt"
	"net/http"

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

	config.basePath = httpx.NormalizeBasePath(config.basePath)
	router := httpx.NewRouter(config.middleware...)

	httpCore := &AegisHTTPCore{
		Responder:    NewResponder(config),
		AuthInstance: a,
		Config:       config,
	}

	registerBaseRoutes(router, httpCore, a.core, config.basePath)

	for _, feature := range a.config.Features {
		featureConfig := feature.PluginHTTPConfig()
		basePath := featureConfig.BasePath
		override := config.overrides[string(feature.Name())]
		if override != nil && override.BasePath != "" {
			basePath = override.BasePath
		}

		normalizedBasePath := config.basePath + httpx.NormalizeBasePath(basePath)
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

func registerBaseRoutes(router *httpx.Router, httpCore *AegisHTTPCore, core *AegisCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group:         router.Group(basePath),
		AegisHTTPCore: httpCore,
	}
	api := NewAegisAPI(httpCore, core)
	api.RegisterRoutes(routeBuilder)
}
