package aegis

import (
	"errors"
)

var (
	ErrDatabaseAdapterRequired = errors.New("database adapter is required")
	ErrPluginNotFound          = errors.New("plugin not found")
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
	ErrInvalidConfiguration    = errors.New("invalid configuration")
)
