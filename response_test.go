package limen

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestResponder(t *testing.T) *Responder {
	t.Helper()
	cfg := NewDefaultHTTPConfig()
	cm := newCookieManager(cfg.cookieConfig, testSecret)
	return newResponder(cfg, cm, false)
}

func TestResponder_JSON(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.JSON(w, req, http.StatusOK, map[string]any{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"key":"value"`)
}

func TestResponder_JSON_StringMessage(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.JSON(w, req, http.StatusOK, "success")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"success"`)
}

func TestResponder_Error(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.Error(w, req, NewLimenError("something went wrong", http.StatusBadRequest, nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"message":"something went wrong"`)
}

func TestResponder_Error_GenericError(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.Error(w, req, errors.New("generic error"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"generic error"`)
}

func TestToLimenError_LimenError(t *testing.T) {
	t.Parallel()

	original := NewLimenError("bad request", http.StatusBadRequest, nil)
	result := ToLimenError(original)

	assert.Equal(t, http.StatusBadRequest, result.Status())
	assert.Equal(t, "bad request", result.Error())
}

func TestToLimenError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "Returns Limen Errors As Is", err: ErrRecordNotFound, want: ErrRecordNotFound.Status()},
		{name: "Returns Generic Errors As Internal Server Error", err: errors.New("something"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLimenError(tt.err)
			assert.Equal(t, tt.want, result.Status())
		})
	}
}

type sessionExtenderPlugin struct {
	testPlugin
}

func (p *sessionExtenderPlugin) ExtendSessionResponse(_ context.Context, _ *User, session *Session) SessionExtension {
	extension := SessionExtension{User: map[string]any{"tag": "extended"}}
	if session != nil {
		extension.Session = map[string]any{"token": session.Token}
	}
	return extension
}

func TestResponder_SessionResponse_Extends(t *testing.T) {
	t.Parallel()

	transformer := WithHTTPSessionTransformer(func(user map[string]any, _ *SessionResult) (map[string]any, error) {
		return map[string]any{"user": map[string]any{"tag": user["tag"]}}, nil
	})

	testData := []struct {
		name          string
		opts          []HTTPConfigOption
		sessionResult *SessionResult
		wantToken     string
	}{
		{name: "current session", wantToken: "current"},
		{name: "issued session", sessionResult: &SessionResult{Session: &Session{Token: "issued"}}, wantToken: "issued"},
		{name: "session transformer", opts: []HTTPConfigOption{transformer}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := newTestLimen(t, &sessionExtenderPlugin{})
			user, err := l.core.DBAction.FindUserByID(t.Context(), seedUser(t, l, "ext@test.com"))
			require.NoError(t, err)
			cfg := NewDefaultHTTPConfig(tt.opts...)
			rs := newResponder(cfg, newCookieManager(cfg.cookieConfig, testSecret), false)
			ctx := context.WithValue(t.Context(), contextKeyActiveSession{}, &ValidatedSession{User: user, Session: &Session{Token: "current"}})
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
			w := httptest.NewRecorder()

			require.NoError(t, rs.SessionResponse(w, req, l.core, &AuthenticationResult{User: user}, tt.sessionResult))

			var body struct {
				User    map[string]any `json:"user"`
				Session map[string]any `json:"session"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "extended", body.User["tag"])
			if tt.wantToken != "" {
				assert.Equal(t, tt.wantToken, body.Session["token"])
			}
		})
	}
}
