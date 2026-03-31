package credentialpassword

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrInvalidCredential         = limen.NewLimenError("invalid credential", http.StatusUnauthorized, nil)
	ErrInvalidPassword           = limen.NewLimenError("invalid password", http.StatusUnauthorized, nil)
	ErrEmailNotFound             = limen.NewLimenError("email not found", http.StatusNotFound, nil)
	ErrEmailRequired             = limen.NewLimenError("email is required", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequired          = limen.NewLimenError("password is required", http.StatusUnprocessableEntity, nil)
	ErrEmailAlreadyExists        = limen.NewLimenError("email already exists", http.StatusConflict, nil)
	ErrPasswordTooShort          = limen.NewLimenError("password is too short", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequiresUppercase = limen.NewLimenError("password requires uppercase letters", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequiresNumbers   = limen.NewLimenError("password requires numbers", http.StatusUnprocessableEntity, nil)
	ErrPasswordRequiresSymbols   = limen.NewLimenError("password requires symbols", http.StatusUnprocessableEntity, nil)
	ErrResetTokenInvalid         = limen.NewLimenError("invalid or expired token. Please request a new one.", http.StatusBadRequest, nil)
	ErrInvalidCurrentPassword    = limen.NewLimenError("current password is invalid", http.StatusUnauthorized, nil)
	ErrEmailAlreadyVerified      = limen.NewLimenError("email already verified", http.StatusConflict, nil)
	ErrUsernameAlreadyExists     = limen.NewLimenError("username already exists", http.StatusConflict, nil)
	ErrUsernameRequired          = limen.NewLimenError("username is required", http.StatusUnprocessableEntity, nil)
	ErrUsernameTooShort          = limen.NewLimenError("username is too short", http.StatusUnprocessableEntity, nil)
	ErrUsernameTooLong           = limen.NewLimenError("username is too long", http.StatusUnprocessableEntity, nil)
	ErrUsernameInvalidFormat     = limen.NewLimenError("username contains invalid characters", http.StatusUnprocessableEntity, nil)
	ErrPasswordNotSet            = limen.NewLimenError("password is not set", http.StatusForbidden, nil)
	ErrPasswordAlreadySet        = limen.NewLimenError("password is already set", http.StatusForbidden, nil)
	ErrUsernameNotEnabled        = limen.NewLimenError("username support is not enabled", http.StatusBadRequest, nil)
)
