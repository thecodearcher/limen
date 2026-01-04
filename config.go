package aegis

import "fmt"

// Config is the main configuration struct for the aegis library
type Config struct {
	Database DatabaseAdapter
	Features []Feature
	Schema   *SchemaConfig
	Session  *sessionConfig
	HTTP     *httpConfig
	CLI      *CLIConfig // Optional: CLI support configuration
}

// CLIConfig contains configuration for CLI tool support
// When enabled, discovered schemas are serialized to a JSON file that the CLI can read directly
type CLIConfig struct {
	Enabled    bool   // Whether CLI support is enabled
	Models     string // Path to the models directory (default: "./models")
	Migrations string // Path to the migrations directory (default: "./migrations")
}

func (c *Config) validate() error {
	if c.Database == nil {
		return ErrDatabaseAdapterRequired
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
