package aegis

import (
	"github.com/thecodearcher/aegis/schemas"
)

// Config is the main configuration struct for the aegis library
type Config struct {
	Database DatabaseAdapter
	Features []Feature
	Schema   schemas.Config
	JWT      *jWTConfig
	Session  *SessionConfig
}

func (c *Config) validate() error {
	if c.Database == nil {
		return ErrDatabaseAdapterRequired
	}

	if c.JWT == nil {
		c.JWT = NewDefaultJWTConfig()
	}

	if c.Session == nil {
		c.Session = NewDefaultSessionConfig()
	}

	if err := c.Session.validate(); err != nil {
		return err
	}

	if err := c.JWT.validate(); err != nil {
		return err
	}

	return nil
}
