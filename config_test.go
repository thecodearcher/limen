package limen

import (
	"testing"
)

func TestConfig_DefaultValues(t *testing.T) {
	config := &Config{
		Database: nil,
	}

	if config.Database != nil {
		t.Errorf("Expected adapter to be nil, got %v", config.Database)
	}
}
