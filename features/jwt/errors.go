package jwt

import "errors"

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
