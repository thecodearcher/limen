package twofactor

import (
	"net/http"

	"github.com/thecodearcher/aegis"
)

var (
	ErrTwoFactorAlreadyEnabled               = aegis.NewAegisError("two factor already enabled", http.StatusConflict, nil)
	ErrInvalidCode                           = aegis.NewAegisError("invalid code", http.StatusUnauthorized, nil)
	ErrTwoFactorNotEnabled                   = aegis.NewAegisError("two factor not enabled", http.StatusUnauthorized, nil)
	ErrInvalidPassword                       = aegis.NewAegisError("invalid password", http.StatusUnauthorized, nil)
	ErrPasswordRequired                      = aegis.NewAegisError("password is required", http.StatusUnprocessableEntity, nil)
	ErrCredentialPasswordFeatureNotAvailable = aegis.NewAegisError("credential password feature not available", http.StatusServiceUnavailable, nil)
	ErrInvalidBackupCode                     = aegis.NewAegisError("invalid backup code", http.StatusUnauthorized, nil)
	ErrInvalidOTPCode                        = aegis.NewAegisError("invalid OTP code", http.StatusUnauthorized, nil)
	ErrInvalidChallenge                      = aegis.NewAegisError("invalid 2FA challenge", http.StatusUnauthorized, nil)
	ErrChallengeExpired                      = aegis.NewAegisError("2FA challenge has expired", http.StatusUnauthorized, nil)
	ErrChallengeMissing                      = aegis.NewAegisError("2FA challenge is missing", http.StatusUnauthorized, nil)
)
