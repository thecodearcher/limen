package admin

import (
	"fmt"
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrInsufficientPermission    = limen.NewLimenErrorWithCode("permission_denied", "You do not have permission to access this resource", http.StatusForbidden, nil)
	ErrEmailAlreadyExists        = limen.NewLimenErrorWithCode("email_already_exists", "A user with this email already exists", http.StatusConflict, nil)
	ErrCannotDeleteSelf          = limen.NewLimenErrorWithCode("cannot_delete_self", "You cannot delete your own account", http.StatusForbidden, nil)
	ErrCannotRemoveOwnAdminRole  = limen.NewLimenErrorWithCode("cannot_remove_own_admin_role", "You cannot remove your own admin role", http.StatusForbidden, nil)
	ErrCannotManageUser          = limen.NewLimenErrorWithCode("cannot_manage_user", "You cannot manage a user with permissions you do not hold", http.StatusForbidden, nil)
	ErrRoleNotGrantable          = limen.NewLimenErrorWithCode("role_not_grantable", "You cannot assign or remove a role with permissions you do not hold", http.StatusForbidden, nil)
	ErrEmailVerificationDisabled = limen.NewLimenErrorWithCode("email_verification_disabled", "Email verification is not enabled", http.StatusNotImplemented, nil)
	ErrPasswordsNotEnabled       = limen.NewLimenErrorWithCode("passwords_not_enabled", "Password authentication is not enabled", http.StatusNotImplemented, nil)
	ErrBansNotEnabled            = limen.NewLimenErrorWithCode("bans_not_enabled", "User bans are not enabled", http.StatusNotImplemented, nil)
	ErrCannotBanSelf             = limen.NewLimenErrorWithCode("cannot_ban_self", "You cannot ban your own account", http.StatusForbidden, nil)
	ErrInvalidBanExpiry          = limen.NewLimenErrorWithCode("invalid_ban_expiry", "Ban expiry must be in the future", http.StatusUnprocessableEntity, nil)
	ErrUserBanned                = limen.NewLimenErrorWithCode("user_banned", "Your account has been banned", http.StatusForbidden, nil)
	ErrImpersonationNotEnabled   = limen.NewLimenErrorWithCode("impersonation_not_enabled", "Impersonation is not enabled", http.StatusNotImplemented, nil)
	ErrCannotImpersonateSelf     = limen.NewLimenErrorWithCode("cannot_impersonate_self", "You cannot impersonate yourself", http.StatusForbidden, nil)
	ErrAlreadyImpersonating      = limen.NewLimenErrorWithCode("already_impersonating", "You are already impersonating a user", http.StatusConflict, nil)
	ErrNotImpersonating          = limen.NewLimenErrorWithCode("not_impersonating", "This session is not impersonating a user", http.StatusConflict, nil)
)

func newUnknownRoleError(name string) *limen.LimenError {
	return limen.NewLimenErrorWithCode("unknown_role", fmt.Sprintf("role %q is not a configured role", name), http.StatusUnprocessableEntity, nil)
}

func newInvalidFilterError(name, reason string) *limen.LimenError {
	return limen.NewLimenErrorWithCode("invalid_filter", fmt.Sprintf("filter[%s] %s", name, reason), http.StatusUnprocessableEntity, nil)
}
