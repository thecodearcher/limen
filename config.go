package aegis

// Config is the main configuration struct for the aegis library
type Config struct {
	Database DatabaseAdapter
	Features []Feature
	Schema   SchemaConfig
	Session  *sessionConfig
}

func (c *Config) validate() error {
	if c.Database == nil {
		return ErrDatabaseAdapterRequired
	}

	if c.Session == nil {
		c.Session = NewDefaultSessionConfig()
	}

	if err := c.Session.validate(); err != nil {
		return err
	}

	return nil
}
