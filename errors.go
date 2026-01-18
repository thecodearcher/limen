package aegis

import (
	"errors"
	"net/http"
)

type AegisError struct {
	message string
	details any
	status  int
}

var (
	ErrDatabaseAdapterRequired = errors.New("database adapter is required")
	ErrPluginNotFound          = errors.New("plugin not found")
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
	ErrInvalidConfiguration    = errors.New("invalid configuration")
)

// Session-specific errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionInvalid  = errors.New("session is invalid")
)

// Rate limiting errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrRateLimitNotFound = errors.New("rate limit not found")
)

func NewAegisError(message string, status int, details any) *AegisError {
	return &AegisError{message: message, details: details, status: status}
}

func (e AegisError) Error() string {
	return e.message
}

func (e AegisError) Details() any {
	return e.details
}

func (e AegisError) Status() int {
	return e.status
}

func ToAegisError(err error) *AegisError {
	if err, ok := err.(*AegisError); ok {
		return err
	}

	return NewAegisError(err.Error(), http.StatusInternalServerError, err)
}
