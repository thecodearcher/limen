package aegis

// Config is the main configuration struct for the aegis library
type Config struct {
	Database DatabaseAdapter
	Features []Feature
	Schema   SchemaConfig
	JWT      *jWTConfig
}

type SchemaConfig struct {
	User UserSchema
}

func (c *Config) validate() error {
	if c.Database == nil {
		return ErrDatabaseAdapterRequired
	}

	if c.JWT == nil {
		c.JWT = NewDefaultJWTConfig()
	}

	if err := c.JWT.validate(); err != nil {
		return err
	}

	return nil
}
