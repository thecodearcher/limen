package aegis

import (
	"fmt"
	"os"
)

// Config is the main configuration struct for the aegis library
type Config struct {
	BaseURL       string
	Database      DatabaseAdapter
	Plugins       []Plugin
	Schema        *SchemaConfig
	Session       *sessionConfig
	HTTP          *httpConfig
	CLI           *CLIConfig
	SigningSecret []byte
}

// CLIConfig contains configuration for CLI tool support
// When enabled, discovered schemas are serialized to a JSON file that the CLI can read directly
type CLIConfig struct {
	Enabled bool
}

func (c *Config) validate() error {
	if c.BaseURL == "" {
		c.BaseURL = "http://localhost:8080"
	}

	if c.Database == nil {
		return ErrDatabaseAdapterRequired
	}

	secret := c.SigningSecret
	if len(secret) == 0 {
		secret = []byte(os.Getenv("AEGIS_SECRET"))
		c.SigningSecret = secret
	}
	if len(secret) == 0 {
		return fmt.Errorf("signing secret is required: set Config.SigningSecret, or AEGIS_SECRET environment variable")
	}

	if len(secret) != 32 {
		return fmt.Errorf("signing secret must be 32 bytes, got %d", len(secret))
	}

	if c.Schema == nil {
		c.Schema = NewDefaultSchemaConfig()
	}

	if c.Session == nil {
		c.Session = NewDefaultSessionConfig()
	}

	if err := c.Session.validate(); err != nil {
		return err
	}

	if c.HTTP == nil {
		c.HTTP = NewDefaultHTTPConfig()
	}

	if err := c.validateHTTP(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateHTTP() error {
	if c.HTTP == nil {
		return nil
	}

	if c.HTTP.rateLimiter != nil {
		c.HTTP.rateLimiter.validate()
	}

	if c.HTTP.cookieConfig != nil && c.HTTP.cookieConfig.crossDomain && len(c.HTTP.trustedOrigins) == 0 {
		return fmt.Errorf("trusted origins are required when cross-domain cookies are enabled")
	}

	return nil
}
