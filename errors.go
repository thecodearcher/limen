package aegis

import (
	"errors"
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

// JWT-specific errors
var (
	ErrTokenExpired                 = errors.New("token has expired")
	ErrTokenInvalid                 = errors.New("token is invalid")
	ErrJWTInvalidAlgorithm          = errors.New("invalid JWT algorithm")
	ErrJWTMissingSecret             = errors.New("JWT HMAC secret is required")
	ErrJWTMissingPrivateKey         = errors.New("JWT private key is required")
	ErrJWTMissingPublicKey          = errors.New("JWT public key is required")
	ErrJWTInvalidDuration           = errors.New("JWT token duration must be positive")
	ErrJWTInvalidRefreshDuration    = errors.New("JWT refresh token duration must be greater than access token duration")
	ErrJWTInvalidIssuer             = errors.New("JWT issuer is required")
	ErrJWTInvalidPrivateKeyConflict = errors.New("either private key path or PEM can be provided, not both")
	ErrJWTInvalidPublicKeyConflict  = errors.New("either public key path or PEM can be provided, not both")
	ErrJWTSubjectFieldConflict      = errors.New("subject field and subject value cannot be provided at the same time")
)

// Session-specific errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionInvalid  = errors.New("session is invalid")
)

type AegisErrorImpl struct {
	message string
	details any
	status  int
}

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
