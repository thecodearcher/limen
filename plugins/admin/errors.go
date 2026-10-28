package admin

import (
	"fmt"
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrInsufficientPermission = limen.NewLimenErrorWithCode("permission_denied", "You do not have permission to access this resource", http.StatusForbidden, nil)
	ErrEmailAlreadyExists     = limen.NewLimenErrorWithCode("email_already_exists", "A user with this email already exists", http.StatusConflict, nil)
	ErrCannotDeleteSelf       = limen.NewLimenErrorWithCode("cannot_delete_self", "You cannot delete your own account", http.StatusForbidden, nil)
	ErrLastAdmin              = limen.NewLimenErrorWithCode("last_admin", "The last admin cannot be removed", http.StatusConflict, nil)
)

func newInvalidFilterError(name, reason string) *limen.LimenError {
	return limen.NewLimenErrorWithCode("invalid_filter", fmt.Sprintf("filter[%s] %s", name, reason), http.StatusUnprocessableEntity, nil)
}
