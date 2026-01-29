package aegis

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("users"))
	}, "users", nil)

	router.AddRoute("GET", "/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + id))
	}, "user", nil)

	router.AddRoute("GET", "/users/:id/posts", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + id + "-posts"))
	}, "user-posts", nil)

	tests := []struct {
		method     string
		path       string
		statusCode int
		body       string
	}{
		{"GET", "/users", http.StatusOK, "users"},
		{"GET", "/users/123", http.StatusOK, "user-123"},
		{"GET", "/users/123/posts", http.StatusOK, "user-123-posts"},
		{"GET", "/users/123/posts/456", http.StatusNotFound, ""},
		{"GET", "/nonexistent", http.StatusNotFound, ""},
		{"POST", "/users", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, w.Body.String())
			}
		})
	}
}

func TestRouterHEADFallback(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}, "test", nil)

	req := httptest.NewRequest("HEAD", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouterPatternMatching(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/api/v1/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + id))
	}, "user", nil)

	tests := []struct {
		path       string
		statusCode int
		body       string
	}{
		{"/api/v1/users/123", http.StatusOK, "user-123"},
		{"/api/v1/users/abc", http.StatusOK, "user-abc"},
		{"/api/v1/users/123/extra", http.StatusNotFound, ""},
		{"/api/v1/users", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, w.Body.String())
			}
		})
	}
}

func TestGetParams(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/test/:param1/:param2", func(w http.ResponseWriter, r *http.Request) {
		params := GetParams(r)
		param1 := GetParam(r, "param1")
		param2 := GetParam(r, "param2")

		if len(params) != 2 {
			t.Errorf("expected 2 params, got %d", len(params))
		}

		if param1 != "value1" {
			t.Errorf("expected param1=value1, got %s", param1)
		}

		if param2 != "value2" {
			t.Errorf("expected param2=value2, got %s", param2)
		}

		w.WriteHeader(http.StatusOK)
	}, "test", nil)

	req := httptest.NewRequest("GET", "/test/value1/value2", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthRoutes(t *testing.T) {
	router := NewRouter()

	router.AddRoute("POST", "/signin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("signin"))
	}, "signin", nil)

	router.AddRoute("POST", "/signup", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("signup"))
	}, "signup", nil)

	router.AddRoute("POST", "/signout", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("signout"))
	}, "signout", nil)

	router.AddRoute("GET", "/session", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("session"))
	}, "session", nil)

	router.AddRoute("GET", "/session/:id", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("session-" + id))
	}, "session-by-id", nil)

	tests := []struct {
		method     string
		path       string
		statusCode int
		body       string
	}{
		{"POST", "/signin", http.StatusOK, "signin"},
		{"POST", "/signup", http.StatusOK, "signup"},
		{"POST", "/signout", http.StatusOK, "signout"},
		{"GET", "/session", http.StatusOK, "session"},
		{"GET", "/session/abc123", http.StatusOK, "session-abc123"},
		{"GET", "/session/abc123/extra", http.StatusNotFound, ""},
		{"POST", "/session", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, w.Body.String())
			}
		})
	}
}

func TestMultipleParameters(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/oauth/:provider/callback/:token", func(w http.ResponseWriter, r *http.Request) {
		provider := GetParam(r, "provider")
		token := GetParam(r, "token")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("oauth-" + provider + "-callback-" + token))
	}, "oauth-callback", nil)

	router.AddRoute("GET", "/api/:version/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		version := GetParam(r, "version")
		userID := GetParam(r, "id")
		postID := GetParam(r, "postId")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api-" + version + "-users-" + userID + "-posts-" + postID))
	}, "user-post", nil)

	tests := []struct {
		method     string
		path       string
		statusCode int
		body       string
	}{
		{"GET", "/oauth/google/callback/abc123", http.StatusOK, "oauth-google-callback-abc123"},
		{"GET", "/oauth/github/callback/def456", http.StatusOK, "oauth-github-callback-def456"},
		{"GET", "/api/v1/users/123/posts/456", http.StatusOK, "api-v1-users-123-posts-456"},
		{"GET", "/api/v2/users/789/posts/101", http.StatusOK, "api-v2-users-789-posts-101"},
		{"GET", "/oauth/google/callback", http.StatusNotFound, ""},
		{"GET", "/oauth/google/callback/abc123/extra", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, w.Body.String())
			}
		})
	}
}

func TestPerMethodHandlers(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET handler"))
	}, "test-get", nil)

	router.AddRoute("POST", "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("POST handler"))
	}, "test-post", nil)

	router.AddRoute("PUT", "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PUT handler"))
	}, "test-put", nil)

	tests := []struct {
		method     string
		path       string
		statusCode int
		body       string
	}{
		{"GET", "/test", http.StatusOK, "GET handler"},
		{"POST", "/test", http.StatusOK, "POST handler"},
		{"PUT", "/test", http.StatusOK, "PUT handler"},
		{"DELETE", "/test", http.StatusNotFound, ""},
		{"PATCH", "/test", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, w.Body.String())
			}
		})
	}
}

func TestExactFastPath(t *testing.T) {
	router := NewRouter()

	router.AddRoute("GET", "/static", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("static"))
	}, "static", nil)

	router.AddRoute("POST", "/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api-users"))
	}, "api-users", nil)

	router.AddRoute("GET", "/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + id))
	}, "user", nil)

	tests := []struct {
		method     string
		path       string
		statusCode int
		body       string
	}{
		{"GET", "/static", http.StatusOK, "static"},
		{"POST", "/api/users", http.StatusOK, "api-users"},
		{"GET", "/users/123", http.StatusOK, "user-123"},
		{"GET", "/nonexistent", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, w.Body.String())
			}
		})
	}
}

func TestRouterPerformance(t *testing.T) {
	router := NewRouter()

	for i := range 1000 {
		path := "/api/v1/users/" + string(rune('a'+i%26)) + string(rune('0'+i%10))
		router.AddRoute("GET", path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}, RouteID("route-"+string(rune(i))), nil)
	}

	// Test that lookup is still fast
	req := httptest.NewRequest("GET", "/api/v1/users/a0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouterMiddleware(t *testing.T) {
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

	router := NewRouter(globalMW)

	router.AddRoute("GET", "/test", func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}, "test", nil, routeMW1, routeMW2)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	executionOrder = []string{}
	router.ServeHTTP(w, req)

	expectedOrder := []string{"global", "route1", "route2", "handler"}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("expected %d middleware calls, got %d", len(expectedOrder), len(executionOrder))
	}
	for i, expected := range expectedOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("expected execution order[%d] = %q, got %q", i, expected, executionOrder[i])
		}
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouterMiddlewareWithParams(t *testing.T) {
	var middlewareCalled bool

	routeMW := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	router := NewRouter()

	router.AddRoute("GET", "/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user-" + id))
	}, "user", nil, routeMW)

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	middlewareCalled = false
	router.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("middleware was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "user-123" {
		t.Errorf("expected body %q, got %q", "user-123", w.Body.String())
	}
}
