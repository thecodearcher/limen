package oauth

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/thecodearcher/aegis"
)

// CallbackError represents a structured OAuth error returned by the authorization
// server in the callback query string (RFC 6749 Section 4.1.2.1).
type CallbackError struct {
	Code        string // required – e.g. "access_denied", "invalid_scope"
	Description string // optional – human-readable explanation
}

func (e *CallbackError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// ToAegisError converts the provider callback error into an AegisError that
// carries the structured OAuth fields in its Details so the handler layer can
// forward them in redirects or JSON responses.
func (e *CallbackError) ToAegisError() *aegis.AegisError {
	details := map[string]string{
		"code":              e.Code,
		"error_description": e.Description,
	}
	msg := e.Description
	if msg == "" {
		msg = e.Code
	}
	return aegis.NewAegisError(msg, http.StatusBadRequest, details)
}

// callbackErrorFromQuery extracts an OAuth error from callback query parameters.
// Returns nil when the "error" param is absent (i.e. no provider error).
func callbackErrorFromQuery(q url.Values) *CallbackError {
	code := q.Get("error")
	if code == "" {
		return nil
	}
	return &CallbackError{
		Code:        code,
		Description: q.Get("error_description"),
	}
}

// appendOAuthErrorParams appends error and error_description query parameters
// to the given URL, skipping empty values.
func appendOAuthErrorParams(rawURL string, code, description string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	if code != "" {
		q.Set("error", code)
	}
	if description != "" {
		q.Set("error_description", description)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
