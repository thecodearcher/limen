package limen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func serveRequest(t *testing.T, router *Router, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRouter(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil)

	router.AddRoute(MethodGET, "/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("users"))
	}, "users", nil)

	router.AddRoute(MethodGET, "/users/:id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + GetParam(r, "id")))
	}, "user", nil)

	router.AddRoute(MethodGET, "/users/:id/posts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + GetParam(r, "id") + "-posts"))
	}, "user-posts", nil)

	router.AddRoute(MethodGET, "/articles/:slug", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("article-" + GetParam(r, "slug")))
	}, "article", nil)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "static route", method: http.MethodGet, path: "/users", wantStatus: http.StatusOK, wantBody: "users"},
		{name: "parameterized route", method: http.MethodGet, path: "/users/123", wantStatus: http.StatusOK, wantBody: "user-123"},
		{name: "alpha param value", method: http.MethodGet, path: "/articles/hello-world", wantStatus: http.StatusOK, wantBody: "article-hello-world"},
		{name: "nested parameterized route", method: http.MethodGet, path: "/users/123/posts", wantStatus: http.StatusOK, wantBody: "user-123-posts"},
		{name: "extra segment returns 404", method: http.MethodGet, path: "/users/123/posts/456", wantStatus: http.StatusNotFound},
		{name: "nonexistent path returns 404", method: http.MethodGet, path: "/nonexistent", wantStatus: http.StatusNotFound},
		{name: "wrong method returns 404", method: http.MethodPost, path: "/users", wantStatus: http.StatusNotFound},
		{name: "prefix of param route returns 404", method: http.MethodGet, path: "/articles", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := serveRequest(t, router, tt.method, tt.path)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestRouterHEADWithoutRegisteredHandler(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil)
	router.AddRoute(MethodGET, "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}, "test", nil)

	w := serveRequest(t, router, http.MethodHead, "/test")

	assert.Equal(t, http.StatusNotFound, w.Code, "HEAD should not fall back to GET handler")
}

func TestGetParams(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil)

	var capturedParams map[string]string
	var capturedParam1, capturedParam2 string

	router.AddRoute(MethodGET, "/test/:param1/:param2", func(w http.ResponseWriter, r *http.Request) {
		capturedParams = GetParams(r)
		capturedParam1 = GetParam(r, "param1")
		capturedParam2 = GetParam(r, "param2")
		w.WriteHeader(http.StatusOK)
	}, "test", nil)

	w := serveRequest(t, router, http.MethodGet, "/test/value1/value2")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, capturedParams, 2)
	assert.Equal(t, "value1", capturedParam1)
	assert.Equal(t, "value2", capturedParam2)
}

func TestMultipleParameters(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil)

	router.AddRoute(MethodGET, "/oauth/:provider/callback/:token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("oauth-" + GetParam(r, "provider") + "-callback-" + GetParam(r, "token")))
	}, "oauth-callback", nil)

	router.AddRoute(MethodGET, "/api/:version/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api-" + GetParam(r, "version") + "-users-" + GetParam(r, "id") + "-posts-" + GetParam(r, "postId")))
	}, "user-post", nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "two params", path: "/oauth/google/callback/abc123", wantStatus: http.StatusOK, wantBody: "oauth-google-callback-abc123"},
		{name: "three params", path: "/api/v1/users/123/posts/456", wantStatus: http.StatusOK, wantBody: "api-v1-users-123-posts-456"},
		{name: "missing last param returns 404", path: "/oauth/google/callback", wantStatus: http.StatusNotFound},
		{name: "extra segment returns 404", path: "/oauth/google/callback/abc123/extra", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := serveRequest(t, router, http.MethodGet, tt.path)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestPerMethodHandlers(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil)

	router.AddRoute(MethodGET, "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET handler"))
	}, "test-get", nil)

	router.AddRoute(MethodPOST, "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("POST handler"))
	}, "test-post", nil)

	router.AddRoute(MethodPUT, "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PUT handler"))
	}, "test-put", nil)

	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{name: "GET handler", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: "GET handler"},
		{name: "POST handler", method: http.MethodPost, wantStatus: http.StatusOK, wantBody: "POST handler"},
		{name: "PUT handler", method: http.MethodPut, wantStatus: http.StatusOK, wantBody: "PUT handler"},
		{name: "DELETE unregistered", method: http.MethodDelete, wantStatus: http.StatusNotFound},
		{name: "PATCH unregistered", method: http.MethodPatch, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := serveRequest(t, router, tt.method, "/test")

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestRouterMiddlewareExecutionOrder(t *testing.T) {
	t.Parallel()

	var executionOrder []string

	globalMW := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "global")
			next.ServeHTTP(w, r)
		})
	})

	routeMW1 := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "route1")
			next.ServeHTTP(w, r)
		})
	})

	routeMW2 := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "route2")
			next.ServeHTTP(w, r)
		})
	})

	router := NewRouter(nil, globalMW)
	router.AddRoute(MethodGET, "/test", func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}, "test", nil, routeMW1, routeMW2)

	executionOrder = nil
	w := serveRequest(t, router, http.MethodGet, "/test")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"global", "route1", "route2", "handler"}, executionOrder)
}

func TestRouterMiddlewareWithParams(t *testing.T) {
	t.Parallel()

	var middlewareCalled bool

	routeMW := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	router := NewRouter(nil)
	router.AddRoute(MethodGET, "/users/:id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + GetParam(r, "id")))
	}, "user", nil, routeMW)

	middlewareCalled = false
	w := serveRequest(t, router, http.MethodGet, "/users/123")

	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-123", w.Body.String())
}
