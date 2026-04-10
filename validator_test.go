package limen

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"empty string", "", true},
		{"nil value", nil, true},
		{"whitespace only", "   ", true},
		{"valid string", "John", false},
		{"int value", 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.RequiredString("field", tt.value)
			err := v.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "field")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		min     int
		wantErr bool
	}{
		{"too short", "abc", 5, true},
		{"exact length", "abcde", 5, false},
		{"longer than min", "johndoe", 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.MinLength("field", tt.value, tt.min)
			if tt.wantErr {
				assert.Error(t, v.Validate())
			} else {
				assert.NoError(t, v.Validate())
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		max     int
		wantErr bool
	}{
		{"too long", "thisiswaytoolong", 10, true},
		{"within limit", "John", 10, false},
		{"exact limit", "1234567890", 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.MaxLength("field", tt.value, tt.max)
			if tt.wantErr {
				assert.Error(t, v.Validate())
			} else {
				assert.NoError(t, v.Validate())
			}
		})
	}
}

func TestEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"invalid email", "invalid-email", true},
		{"valid email", "valid@example.com", false},
		{"empty string skipped", "", false},
		{"nil skipped", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.Email("email", tt.value)
			if tt.wantErr {
				assert.Error(t, v.Validate())
			} else {
				assert.NoError(t, v.Validate())
			}
		})
	}
}

func TestChaining(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	v.RequiredString("email", "").
		Email("email2", "invalid-email").
		MinLength("password", "abc", 8).
		MaxLength("username", "toolongusername", 10)

	err := v.Validate()
	require.Error(t, err)

	errors := err.(*Errors)
	assert.Len(t, errors.GetErrors(), 4)
}

func TestIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		allowed []string
		wantErr bool
	}{
		{"not in list", "pending", []string{"active", "inactive"}, true},
		{"in list", "admin", []string{"admin", "user", "guest"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.In("field", tt.value, tt.allowed)
			if tt.wantErr {
				assert.Error(t, v.Validate())
			} else {
				assert.NoError(t, v.Validate())
			}
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		substr  string
		wantErr bool
	}{
		{"missing substr", "mypassword123", "!", true},
		{"has substr", "ABC123", "123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.Contains("field", tt.value, tt.substr)
			if tt.wantErr {
				assert.Error(t, v.Validate())
			} else {
				assert.NoError(t, v.Validate())
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		chars   string
		wantErr bool
	}{
		{"missing chars", "mypassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", true},
		{"has chars", "MyPassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ContainsAny("field", tt.value, tt.chars)
			if tt.wantErr {
				assert.Error(t, v.Validate())
			} else {
				assert.NoError(t, v.Validate())
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	t.Parallel()

	emailPasswordValidator := func(v *Validator, d map[string]any) *Validator {
		email, _ := d["email"].(string)
		password, _ := d["password"].(string)
		return v.RequiredString("email", email).
			Email("email", email).
			RequiredString("password", password).
			MinLength("password", password, 8)
	}

	t.Run("valid data", func(t *testing.T) {
		req := newValidatorTestRequest(t, `{"email":"test@example.com","password":"secret123"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, emailPasswordValidator)

		require.NotNil(t, data)
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, "secret123", data["password"])
	})

	t.Run("validation error", func(t *testing.T) {
		req := newValidatorTestRequest(t, `{"email":"invalid","password":"short"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, emailPasswordValidator)

		assert.Nil(t, data)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := newValidatorTestRequest(t, `{"email":"test@example.com"`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, func(v *Validator, d map[string]any) *Validator {
			return v
		})

		assert.Nil(t, data)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required field", func(t *testing.T) {
		req := newValidatorTestRequest(t, `{"email":"test@example.com"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, emailPasswordValidator)

		assert.Nil(t, data)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

// newValidatorTestRequest creates a POST request with the JSON body already
// parsed into the request context, matching what the router middleware does.
func newValidatorTestRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = parseAndStoreBody(req)
	return req
}
