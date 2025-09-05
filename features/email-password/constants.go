package emailpassword

import "errors"

const (
	defaultMinPasswordLength        = 4
	defaultPasswordRequireUppercase = true
	defaultPasswordRequireNumbers   = true
	defaultPasswordRequireSymbols   = false
)

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

const (
	PasswordResetAction     = "password_reset"
	EmailVerificationAction = "email_verification"
)
