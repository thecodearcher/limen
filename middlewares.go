package aegis

import (
	"context"
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type contextKeyActiveSession struct{}

func GetCurrentSessionFromCtx(r *http.Request) (*AegisSession, error) {
	if currentSession, ok := r.Context().Value(contextKeyActiveSession{}).(*AegisSession); ok && currentSession != nil {
		return currentSession, nil
	}
	return nil, ErrSessionNotFound
}

// MiddlewareRequireSession is a middleware that requires a session to be present in the request context.
//
// When a session is present, it is added to the request context and can be accessed using the GetCurrentSessionFromCtx() function.
func (httpCore *AegisHTTPCore) MiddlewareRequireSession() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := httpCore.AuthInstance.GetSession(r)
			if err != nil {
				httpCore.AuthInstance.sessionManager.RevokeAllCookies(w)
				httpCore.Responder.Error(w, r, NewAegisError(err.Error(), http.StatusUnauthorized, nil))
				return
			}

			if session.SessionExtensionResult != nil && session.SessionExtensionResult.Cookie != nil {
				http.SetCookie(w, session.SessionExtensionResult.Cookie)
			}

			if session.SessionExtensionResult != nil && session.SessionExtensionResult.Token != "" {
				w.Header().Set("Set-Aegis-Token", session.SessionExtensionResult.Token)
			}

			r = r.WithContext(context.WithValue(r.Context(), contextKeyActiveSession{}, &AegisSession{
				User:    session.User,
				Session: session.Session,
			}))

			next.ServeHTTP(w, r)
		})
	}
}
