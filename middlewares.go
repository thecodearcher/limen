package aegis

import (
	"context"
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type contextKeyActiveSession struct{}

func GetCurrentSessionFromCtx(r *http.Request) (*AegisSession, error) {
	return r.Context().Value(contextKeyActiveSession{}).(*AegisSession), nil
}

// MiddlewareRequireSession is a middleware that requires a session to be present in the request context.
//
// When a session is present, it is added to the request context and can be accessed using the GetCurrentSessionFromCtx() function.
func (httpCore *AegisHTTPCore) MiddlewareRequireSession() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := httpCore.AuthInstance.sessionManager.ValidateSession(r.Context(), r)
			if err != nil {
				httpCore.Responder.Error(w, r, NewAegisError(err.Error(), http.StatusUnauthorized, nil))
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), contextKeyActiveSession{}, &AegisSession{
				User:    session.User,
				Session: session.Session,
			}))

			next.ServeHTTP(w, r)
		})
	}
}
