package oauth

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallbackError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      CallbackError
		expected string
	}{
		{
			name:     "code and description",
			err:      CallbackError{Code: "access_denied", Description: "user declined"},
			expected: "access_denied: user declined",
		},
		{
			name:     "code only",
			err:      CallbackError{Code: "access_denied"},
			expected: "access_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestCallbackError_ToLimenError(t *testing.T) {
	t.Parallel()

	cbErr := &CallbackError{Code: "invalid_scope", Description: "scope not allowed"}
	limenErr := cbErr.ToLimenError()

	assert.Equal(t, "scope not allowed", limenErr.Error())
	assert.Equal(t, http.StatusBadRequest, limenErr.Status())
	details, ok := limenErr.Details().(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "invalid_scope", details["code"])
	assert.Equal(t, "scope not allowed", details["error_description"])
}

func TestCallbackErrorFromQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      url.Values
		expectNil  bool
		expectCode string
		expectDesc string
	}{
		{
			name:      "no error param",
			query:     url.Values{"code": {"abc"}},
			expectNil: true,
		},
		{
			name:       "error with description",
			query:      url.Values{"error": {"access_denied"}, "error_description": {"user said no"}},
			expectCode: "access_denied",
			expectDesc: "user said no",
		},
		{
			name:       "error without description",
			query:      url.Values{"error": {"server_error"}},
			expectCode: "server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := callbackErrorFromQuery(tt.query)
			if tt.expectNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.expectCode, result.Code)
			assert.Equal(t, tt.expectDesc, result.Description)
		})
	}
}

func TestAppendOAuthErrorParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rawURL      string
		code        string
		description string
		wantQuery   map[string]string
		wantAbsent  []string
	}{
		{
			name:        "both code and description",
			rawURL:      "https://example.com/callback",
			code:        "access_denied",
			description: "user declined",
			wantQuery: map[string]string{
				"error":             "access_denied",
				"error_description": "user declined",
			},
		},
		{
			name:   "code only",
			rawURL: "https://example.com/callback",
			code:   "server_error",
			wantQuery: map[string]string{
				"error": "server_error",
			},
			wantAbsent: []string{"error_description"},
		},
		{
			name:       "empty code and description",
			rawURL:     "https://example.com/callback",
			wantAbsent: []string{"error", "error_description"},
		},
		{
			name:   "preserves existing query params",
			rawURL: "https://example.com/callback?existing=yes",
			code:   "err",
			wantQuery: map[string]string{
				"existing": "yes",
				"error":    "err",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := appendOAuthErrorParams(tt.rawURL, tt.code, tt.description)
			parsed, err := url.Parse(result)
			require.NoError(t, err)
			query := parsed.Query()
			for key, want := range tt.wantQuery {
				assert.Equal(t, want, query.Get(key))
			}
			for _, key := range tt.wantAbsent {
				assert.Empty(t, query.Get(key))
			}
		})
	}
}
