package limen

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type bodyContextKey struct{}

// NormalizePath normalizes the base path to start with a slash
func NormalizePath(basePath string) string {
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	basePath = strings.TrimSuffix(basePath, "/")
	return basePath
}

// GetBody extracts the parsed JSON body from the request context
func GetJSONBody(req *http.Request) map[string]any {
	if req == nil {
		return nil
	}

	if body, ok := req.Context().Value(bodyContextKey{}).(map[string]any); ok {
		return body
	}

	return nil
}

// shouldParseBody checks if the request body should be parsed as JSON
func shouldParseBody(req *http.Request) bool {
	if req.Method != "POST" && req.Method != "PUT" && req.Method != "PATCH" {
		return false
	}
	contentType := req.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json") && req.Body != nil
}

// parseJSONBody reads and parses the JSON body from the request
// Returns the parsed body map and the original body bytes for restoration
func parseJSONBody(req *http.Request) (map[string]any, []byte, error) {
	defer req.Body.Close()
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, err
	}

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, bodyBytes, err
	}

	return body, bodyBytes, nil
}

// parseAndStoreBody parses the JSON body if needed and stores it in context
// Returns the updated request and whether body was parsed/stored
func parseAndStoreBody(req *http.Request) (*http.Request, bool) {
	if GetJSONBody(req) != nil {
		return req, false
	}

	if !shouldParseBody(req) {
		return req, false
	}

	body, bodyBytes, err := parseJSONBody(req)
	if err != nil {
		return req, false
	}

	req = req.WithContext(context.WithValue(req.Context(), bodyContextKey{}, body))

	// Restore body for handlers that need to read it
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return req, true
}

func GetCurrentRouteFromContext(ctx context.Context) *Route {
	if route, ok := ctx.Value(currentRouteContextKey{}).(*Route); ok {
		return route
	}
	return nil
}
