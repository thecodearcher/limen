package aegis

import "fmt"

type Aegis struct {
	config *Config
}

func New(config *Config) (*Aegis, error) {
	if config == nil {
		return nil, fmt.Errorf("missing configuration")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Aegis{config: config}, nil
}
