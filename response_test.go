package limen

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestResponder(t *testing.T) *Responder {
	t.Helper()
	cfg := NewDefaultHTTPConfig()
	cm := newCookieManager(cfg.cookieConfig, TestSecret)
	return newResponder(cfg, cm, false)
}

func TestResponder_JSON(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	responder.JSON(w, req, http.StatusOK, map[string]any{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"key":"value"`)
}

func TestResponder_JSON_StringMessage(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	responder.JSON(w, req, http.StatusOK, "success")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"success"`)
}

func TestResponder_Error(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	responder.Error(w, req, NewLimenError("something went wrong", http.StatusBadRequest, nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"message":"something went wrong"`)
}

func TestResponder_Error_GenericError(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
