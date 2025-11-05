// Package aegis provides a framework for building authentication systems.
package aegis

import (
	"fmt"
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
	"github.com/thecodearcher/aegis/schemas"
)

type Aegis struct {
	EmailPassword EmailPasswordFeature
	JWT           TokenGenerator
	config        *Config
}

type AegisCore struct {
	DB      DatabaseAdapter
	Schema  schemas.Config
	JWT     *JwtHandler
	Session *SessionConfig
}

func New(config *Config) (*Aegis, error) {
	if config == nil {
		return nil, fmt.Errorf("missing configuration")
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	jwtHandler, err := newJwtHandler(config.JWT)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwt handler: %w", err)
	}

	aegis := &Aegis{
		JWT:    jwtHandler,
		config: config,
	}
	core := &AegisCore{
		DB:      config.Database,
		Schema:  config.Schema,
		JWT:     jwtHandler,
		Session: config.Session,
	}

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
	config := &HTTPConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.basePath == "" {
		config.basePath = "/auth"
	}

	config.basePath = httpx.NormalizeBasePath(config.basePath)

	router := httpx.NewRouter(config.middleware...)
	responder := NewResponder(config)
	for _, feature := range a.config.Features {
		mount := feature.HTTPMount(a, &responder, config)
		basePath := mount.DefaultBase
		override := config.overrides[string(feature.Name())]
		if override != nil && override.BasePath != "" {
			basePath = httpx.NormalizeBasePath(override.BasePath)
		}

		normalizedBasePath := config.basePath + httpx.NormalizeBasePath(basePath)
		if override != nil && len(override.Middleware) > 0 {
			router.Mount(normalizedBasePath, mount.Handler, override.Middleware...)
		} else {
			router.Mount(normalizedBasePath, mount.Handler)
		}
	}

	return router
}
