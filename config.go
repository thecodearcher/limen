package aegis

// Config is the main configuration struct for the aegis library
type Config struct {
	Database DatabaseAdapter
	Features []Feature
	Schema   SchemaConfig
}

type SchemaConfig struct {
	User UserSchema
}

func (c *Config) Validate() error {
	if c.Database == nil {
		return ErrDatabaseAdapterRequired
	}

	return nil
}
