package limen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestHookContext(t *testing.T) *HookContext {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/signin", http.NoBody)
	w := httptest.NewRecorder()
	responder := newTestResponder(t)
	return &HookContext{
		request:          req,
		response:         w,
		routeID:          "credential-password:sign-in",
		routePattern:     "/auth/signin",
		method:           http.MethodPost,
		path:             "/auth/signin",
		statusCode:       200,
		originalBodyData: map[string]any{"email": "test@example.com", "password": "secret"},
		responder:        responder,
	}
}

func TestHookContext_WriteJSONResponse(t *testing.T) {
	t.Parallel()

	hc := newTestHookContext(t)
	hc.WriteJSONResponse(http.StatusOK, map[string]any{"ok": true})

	w := hc.response.(*httptest.ResponseRecorder)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok":true`)
}

func TestHookContext_WriteErrorResponse(t *testing.T) {
	t.Parallel()

	hc := newTestHookContext(t)
	hc.WriteErrorResponse(NewLimenError("forbidden", http.StatusForbidden, nil))

	w := hc.response.(*httptest.ResponseRecorder)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"forbidden"`)
}

func TestHookContext_SetResponseCookie_Nil(t *testing.T) {
	t.Parallel()

	hc := newTestHookContext(t)
	hc.SetResponseCookie(nil)
	assert.Empty(t, hc.Response().Header().Get("Set-Cookie"))
}

func TestHookContext_RemoveResponseCookie(t *testing.T) {
	t.Parallel()

	hc := newTestHookContext(t)
	http.SetCookie(hc.response, &http.Cookie{Name: "keep", Value: "yes"})
	http.SetCookie(hc.response, &http.Cookie{Name: "remove", Value: "yes"})

	hc.RemoveResponseCookie("remove")

	cookies := hc.Response().Header().Values("Set-Cookie")
	assert.Len(t, cookies, 1)
	assert.Contains(t, cookies[0], "keep=yes")
}

func TestHookContext_GetResponse(t *testing.T) {
	t.Parallel()

	t.Run("nil when response is not a responseWriter", func(t *testing.T) {
		hc := newTestHookContext(t)
		assert.Nil(t, hc.GetResponse())
	})

	t.Run("nil when response is not yet written", func(t *testing.T) {
		rw := &responseWriter{
			ResponseWriter: httptest.NewRecorder(),
			written:        false,
		}
		hc := &HookContext{response: rw}
		assert.Nil(t, hc.GetResponse())
	})

	t.Run("returns data when written", func(t *testing.T) {
		rw := &responseWriter{
			ResponseWriter: httptest.NewRecorder(),
			written:        true,
			statusCode:     http.StatusOK,
			payload:        map[string]any{"user": "test"},
			isError:        false,
		}
		hc := &HookContext{
			response:  rw,
			responder: newTestResponder(t),
		}
		resp := hc.GetResponse()
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.False(t, resp.IsError)
	})
}

func TestHookContext_ModifyResponse(t *testing.T) {
	t.Parallel()

	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		written:        true,
	}
	hc := &HookContext{response: rw}
	hc.ModifyResponse(http.StatusCreated, map[string]any{"modified": true})

	assert.True(t, rw.modified)
	assert.Equal(t, http.StatusCreated, rw.modifiedStatus)
}

func TestHookContext_GetAuthResult(t *testing.T) {
	t.Parallel()

	t.Run("returns auth result from responseWriter", func(t *testing.T) {
		auth := &AuthenticationResult{User: &User{Email: "auth@test.com"}}
		rw := &responseWriter{
			ResponseWriter: httptest.NewRecorder(),
			authResult:     auth,
		}
		hc := &HookContext{response: rw}
		result := hc.GetAuthResult()
		assert.Equal(t, "auth@test.com", result.User.Email)
	})

	t.Run("nil for non-responseWriter", func(t *testing.T) {
		hc := newTestHookContext(t)
		assert.Nil(t, hc.GetAuthResult())
	})
}
