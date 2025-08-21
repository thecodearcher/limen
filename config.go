package aegis

// Config is the main configuration struct for the aegis library
type Config struct {
	Database DatabaseConfig
}

type DatabaseConfig struct {
	Adapter DatabaseAdapter
}

func (c *Config) Validate() error {
	if c.Database.Adapter == nil {
		return ErrDatabaseAdapterRequired
	}

	return nil
}
