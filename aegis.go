// Package aegis provides a framework for building authentication systems.
package aegis

import (
	"fmt"
)

type Aegis struct {
	EmailPassword EmailPasswordFeature
}

type AegisCore struct {
	DB     DatabaseAdapter
	Schema SchemaConfig
}

func New(config *Config) (*Aegis, error) {
	if config == nil {
		return nil, fmt.Errorf("missing configuration")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	aegis := &Aegis{}
	core := &AegisCore{
		DB:     config.Database,
		Schema: config.Schema,
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
