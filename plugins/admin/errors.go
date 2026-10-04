package admin

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrInsufficientPermission = limen.NewLimenErrorWithCode("permission_denied", "You do not have permission to access this resource", http.StatusForbidden, nil)
)
