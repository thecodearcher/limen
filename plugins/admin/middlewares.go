package admin

import (
	"net/http"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (h *adminHandlers) requirePermission(specs ...string) limen.Middleware {
	required := access.P(specs...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := limen.GetCurrentSessionFromCtx(r.Context())
			if err != nil {
				h.responder.Error(w, r, err)
				return
			}

			if err := h.plugin.HasPermission(r.Context(), session.User, required); err != nil {
				h.responder.Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
