package limen

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

type contextKeyActiveSession struct{}

func GetCurrentSessionFromCtx(r *http.Request) (*ValidatedSession, error) {
	if currentSession, ok := r.Context().Value(contextKeyActiveSession{}).(*ValidatedSession); ok && currentSession != nil {
		return currentSession, nil
	}
	return nil, NewLimenError(ErrSessionNotFound.Error(), http.StatusUnauthorized, nil)
}

// MiddlewareRequireSession is a middleware that requires a session to be present in the request context.
//
// When a session is present, it is added to the request context and can be accessed using the GetCurrentSessionFromCtx() function.
func (httpCore *LimenHTTPCore) MiddlewareRequireSession() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := httpCore.authInstance.GetSession(r)
			if err != nil {
				httpCore.core.Cookies().ClearSessionCookie(w)
				httpCore.Responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
				return
			}

			if session.Refreshed != nil {
				if err := httpCore.core.Cookies().SetSessionCookie(w, session.Refreshed); err != nil {
					httpCore.Responder.Error(w, r, err)
					return
				}
			}

			r = r.WithContext(context.WithValue(r.Context(), contextKeyActiveSession{}, &ValidatedSession{
				User:    session.User,
				Session: session.Session,
			}))

			next.ServeHTTP(w, r)
		})
	}
}

func (httpCore *LimenHTTPCore) middlewareCheckOrigin() Middleware {
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

			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}

			if httpCore.IsTrustedOrigin(origin) {
				next.ServeHTTP(w, r)
				return
			}
			httpCore.Responder.Error(w, r, NewLimenError("Origin not allowed", http.StatusForbidden, nil))
		})
	}
}

func (httpCore *LimenHTTPCore) middlewareCSRFProtection() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for GET, HEAD, OPTIONS
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			contentType := strings.Split(r.Header.Get("Content-Type"), ";")[0]
			route := getCurrentRouteFromContext(r.Context())

			if route != nil && route.Metadata != nil && len(route.Metadata.AllowedContentTypes) > 0 {
				if !slices.Contains(route.Metadata.AllowedContentTypes, contentType) {
					httpCore.Responder.Error(w, r, NewLimenError("Content-Type not allowed", http.StatusUnsupportedMediaType, nil))
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

			httpCore.Responder.Error(w, r, NewLimenError("Forbidden", http.StatusForbidden, nil))
		})
	}
}

// middlewareAdditionalFieldsContext stores AdditionalFieldsContext in request context.
// This allows additional fields functions to access request/response data automatically.
func middlewareAdditionalFieldsContext() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := withAdditionalFieldsContext(r.Context(), r, w)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
