package credentialpassword

import (
	"errors"
	"net/http"
)

// Plugin Errors - These errors are more granular and only used within the plugin not the API

var (
	ErrInvalidCredential             = errors.New("invalid credential")
	ErrInvalidPassword               = errors.New("invalid password")
	ErrEmailNotFound                 = errors.New("email not found")
	ErrEmailRequired                 = errors.New("email is required")
	ErrPasswordRequired              = errors.New("password is required")
	ErrEmailAlreadyExists            = errors.New("email already exists")
	ErrPasswordTooShort              = errors.New("password is too short")
	ErrPasswordRequiresUppercase     = errors.New("password requires uppercase letters")
	ErrPasswordRequiresNumbers       = errors.New("password requires numbers")
	ErrPasswordRequiresSymbols       = errors.New("password requires symbols")
	ErrResetTokenRequired            = errors.New("reset token is required")
	ErrResetTokenInvalid             = errors.New("invalid or expired token. Please request a new one.")
	ErrEmailVerificationTokenInvalid = errors.New("invalid or expired email verification token. Please request a new one.")
	ErrInvalidCurrentPassword        = errors.New("current password is invalid")
	ErrEmailAlreadyVerified          = errors.New("email already verified")
	ErrUsernameAlreadyExists         = errors.New("username already exists")
	ErrUsernameRequired              = errors.New("username is required")
	ErrUsernameTooShort              = errors.New("username is too short")
	ErrUsernameTooLong               = errors.New("username is too long")
	ErrUsernameInvalidFormat         = errors.New("username contains invalid characters")
	ErrPasswordReuseNotAllowed       = errors.New("new password must be different from current password")
	ErrPasswordNotSet                = errors.New("password is not set")
)

func errorStatus(err error) int {
	switch err {
	case ErrInvalidCredential:
		return http.StatusUnauthorized
	case ErrEmailAlreadyExists, ErrUsernameAlreadyExists:
		return http.StatusConflict
	case ErrPasswordTooShort,
		ErrPasswordRequiresUppercase,
		ErrPasswordRequiresNumbers,
		ErrPasswordRequiresSymbols,
		ErrUsernameTooShort,
		ErrUsernameTooLong,
		ErrUsernameInvalidFormat,
		ErrEmailRequired,
		ErrPasswordRequired,
		ErrPasswordReuseNotAllowed:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusBadRequest
	}
}
