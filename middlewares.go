package aegis

import (
	"context"
	"net/http"
	"slices"
	"strings"

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
			session, err := httpCore.authInstance.GetSession(r)
			if err != nil {
				httpCore.core.SessionManager.RevokeAllCookies(w)
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

func (httpCore *AegisHTTPCore) middlewareCheckOrigin() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// allow non-mutating methods to pass through
			if r.Method == http.MethodOptions || r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			if len(httpCore.trustedOriginsPatterns) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			if originMatcher(r, httpCore.trustedOriginsPatterns) {
				next.ServeHTTP(w, r)
				return
			}
			httpCore.Responder.Error(w, r, NewAegisError("Origin not allowed", http.StatusForbidden, nil))
		})
	}
}

func (httpCore *AegisHTTPCore) middlewareCSRFProtection() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for GET, HEAD, OPTIONS
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			contentType := strings.Split(r.Header.Get("Content-Type"), ";")[0]
			route := httpx.GetCurrentRouteFromContext(r.Context())

			if route != nil && route.Metadata != nil && len(route.Metadata.AllowedContentTypes) > 0 {
				if !slices.Contains(route.Metadata.AllowedContentTypes, contentType) {
					httpCore.Responder.Error(w, r, NewAegisError("Content-Type not allowed", http.StatusUnsupportedMediaType, nil))
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			if contentType != "" && strings.HasPrefix(contentType, "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			// Check for non-simple headers
			if r.Header.Get("Authorization") != "" || r.Header.Get("X-Requested-With") != "" {
				next.ServeHTTP(w, r)
				return
			}

			httpCore.Responder.Error(w, r, NewAegisError("Forbidden", http.StatusForbidden, nil))
		})
	}
}
