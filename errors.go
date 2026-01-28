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
	ErrRecordNotFound          = errors.New("record not found")
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

// Verification errors
var (
	ErrVerificationTokenInvalid = errors.New("verification token is invalid")
)

func NewAegisError(message string, status int, details any) *AegisError {
	return &AegisError{message: message, details: details, status: status}
}

func (e *AegisError) Error() string {
	return e.message
}

func (e *AegisError) Details() any {
	return e.details
}

func (e *AegisError) Status() int {
	return e.status
}

func ToAegisError(err error) *AegisError {
	var aegisErr *AegisError
	if errors.As(err, &aegisErr) {
		return err.(*AegisError)
	}

	if errors.Is(err, ErrRecordNotFound) {
		return NewAegisError(err.Error(), http.StatusNotFound, err)
	}

	return NewAegisError(err.Error(), http.StatusInternalServerError, err)
}
