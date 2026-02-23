package credentialpassword

import (
	"net/http"

	"github.com/thecodearcher/aegis"
)

var (
	ErrInvalidCredential         = aegis.NewAegisError("invalid credential", http.StatusUnauthorized, nil)
	ErrInvalidPassword           = aegis.NewAegisError("invalid password", http.StatusUnauthorized, nil)
	ErrEmailNotFound             = aegis.NewAegisError("email not found", http.StatusNotFound, nil)
	ErrEmailRequired             = aegis.NewAegisError("email is required", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequired          = aegis.NewAegisError("password is required", http.StatusUnprocessableEntity, nil)
	ErrEmailAlreadyExists        = aegis.NewAegisError("email already exists", http.StatusConflict, nil)
	ErrPasswordTooShort          = aegis.NewAegisError("password is too short", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequiresUppercase = aegis.NewAegisError("password requires uppercase letters", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequiresNumbers   = aegis.NewAegisError("password requires numbers", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequiresSymbols   = aegis.NewAegisError("password requires symbols", http.StatusUnprocessableEntity, nil)
	ErrResetTokenInvalid         = aegis.NewAegisError("invalid or expired token. Please request a new one.", http.StatusBadRequest, nil)
	ErrInvalidCurrentPassword    = aegis.NewAegisError("current password is invalid", http.StatusUnauthorized, nil)
	ErrEmailAlreadyVerified      = aegis.NewAegisError("email already verified", http.StatusConflict, nil)
	ErrUsernameAlreadyExists     = aegis.NewAegisError("username already exists", http.StatusConflict, nil)
	ErrUsernameRequired          = aegis.NewAegisError("username is required", http.StatusUnprocessableEntity, nil)
	ErrUsernameTooShort          = aegis.NewAegisError("username is too short", http.StatusUnprocessableEntity, nil)
	ErrUsernameTooLong           = aegis.NewAegisError("username is too long", http.StatusUnprocessableEntity, nil)
	ErrUsernameInvalidFormat     = aegis.NewAegisError("username contains invalid characters", http.StatusUnprocessableEntity, nil)
	ErrPasswordReuseNotAllowed   = aegis.NewAegisError("new password must be different from current password", http.StatusUnprocessableEntity, nil)
	ErrPasswordNotSet            = aegis.NewAegisError("password is not set", http.StatusForbidden, nil)
	ErrPasswordAlreadySet        = aegis.NewAegisError("password is already set", http.StatusForbidden, nil)
)
