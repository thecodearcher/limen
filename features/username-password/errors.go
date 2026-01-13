package usernamepassword

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/aegis"
)

// Plugin Errors - These errors are more granular and only used within the plugin not the API

var (
	ErrUsernameNotFound         = errors.New("username not found")
	ErrUsernameAlreadyExists    = errors.New("username already exists")
	ErrUsernameRequired         = errors.New("username is required")
	ErrUsernameTooShort         = errors.New("username is too short")
	ErrUsernameTooLong          = errors.New("username is too long")
	ErrUsernameInvalidFormat    = errors.New("username contains invalid characters")
	ErrEmailPasswordNotEnabled  = errors.New("email-password feature must be enabled to use username-password")
)

// API Errors - These errors are used in the API layer to return errors to the client
var (
	ErrAPIInvalidCredentials = aegis.NewAegisError("invalid credentials", http.StatusUnauthorized, nil)
)
