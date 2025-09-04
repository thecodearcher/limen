// Package aegis provides a framework for building authentication systems.
package aegis

import (
	"fmt"

	"github.com/thecodearcher/aegis/schemas"
)

type Aegis struct {
	EmailPassword EmailPasswordFeature
	JWT           TokenGenerator
}

type AegisCore struct {
	DB     DatabaseAdapter
	Schema schemas.Config
	JWT    *JwtHandler
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
		JWT: jwtHandler,
	}
	core := &AegisCore{
		DB:     config.Database,
		Schema: config.Schema,
		JWT:    jwtHandler,
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
