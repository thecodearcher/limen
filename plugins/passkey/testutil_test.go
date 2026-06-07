package passkey

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thecodearcher/limen"
)

func newTestLimenAndPlugin(t *testing.T, opts ...ConfigOption) (*limen.Limen, *passkeyPlugin) {
	t.Helper()
	plugin := New(opts...)
	l, _ := limen.NewTestLimen(t, plugin)
	return l, plugin
}

func newJSONRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
