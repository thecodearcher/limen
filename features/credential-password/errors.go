package credentialpassword

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/aegis"
)

// Plugin Errors - These errors are more granular and only used within the plugin not the API

var (
	ErrInvalidPassword           = errors.New("invalid password")
	ErrEmailNotFound             = errors.New("email not found")
	ErrEmailRequired             = errors.New("email is required")
	ErrPasswordRequired          = errors.New("password is required")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrPasswordTooShort          = errors.New("password is too short")
	ErrPasswordRequiresUppercase = errors.New("password requires uppercase letters")
	ErrPasswordRequiresNumbers   = errors.New("password requires numbers")
	ErrPasswordRequiresSymbols   = errors.New("password requires symbols")
	ErrResetTokenRequired        = errors.New("reset token is required")
	ErrResetTokenInvalid         = errors.New("invalid or expired token. Please request a new one.")
	ErrInvalidCurrentPassword    = errors.New("current password is invalid")
	ErrEmailAlreadyVerified      = errors.New("email already verified")
)

// API Errors - These errors are used in the API layer to return errors to the client
var (
	ErrAPIInvalidCredentials = aegis.NewAegisError("invalid credentials", http.StatusUnauthorized, nil)
)
