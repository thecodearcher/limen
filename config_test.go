package aegis

import (
	"testing"
)

func TestConfig_DefaultValues(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Adapter: nil,
		},
	}

	if config.Database.Adapter != nil {
		t.Errorf("Expected adapter to be nil, got %v", config.Database.Adapter)
	}
}
