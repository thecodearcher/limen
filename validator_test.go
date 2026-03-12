package aegis

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequiredString(t *testing.T) {
	v := NewValidator()
	v.RequiredString("email", "")
	v.RequiredString("name", "John")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for empty email")
	}

	if !strings.Contains(err.Error(), "email") {
		t.Error("Expected error to mention email field")
	}
}

func TestMinLength(t *testing.T) {
	v := NewValidator()
	v.MinLength("password", "abc", 5)
	v.MinLength("username", "johndoe", 5)

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for short password")
	}
}

func TestMaxLength(t *testing.T) {
	v := NewValidator()
	v.MaxLength("username", "thisiswaytoolong", 10)
	v.MaxLength("name", "John", 10)

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for long username")
	}
}

func TestEmail(t *testing.T) {
	v := NewValidator()
	v.Email("email", "invalid-email")
	v.Email("email2", "valid@example.com")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid email")
	}
}

func TestChaining(t *testing.T) {
	v := NewValidator()
	v.RequiredString("email", "").
		Email("email2", "invalid-email").
		MinLength("password", "abc", 8).
		MaxLength("username", "toolongusername", 10)

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation errors")
	}

	errors := err.(*Errors)
	if len(errors.GetErrors()) != 4 {
		t.Errorf("Expected 4 errors, got %d", len(errors.GetErrors()))
	}
}

func TestIn(t *testing.T) {
	v := NewValidator()
	v.In("status", "pending", []string{"active", "inactive"})
	v.In("role", "admin", []string{"admin", "user", "guest"})

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for status")
	}

	// Should not have error for role
	v2 := NewValidator()
	v2.In("role", "admin", []string{"admin", "user", "guest"})
	if err := v2.Validate(); err != nil {
		t.Error("Expected no error for valid role")
	}
}

func TestContains(t *testing.T) {
	v := NewValidator()
	v.Contains("password", "mypassword123", "!")
	v.Contains("code", "ABC123", "123")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for password missing !")
	}

	// Should not have error for code
	v2 := NewValidator()
	v2.Contains("code", "ABC123", "123")
	if err := v2.Validate(); err != nil {
		t.Error("Expected no error for code containing 123")
	}
}

func TestContainsAny(t *testing.T) {
	v := NewValidator()
	v.ContainsAny("password", "mypassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	v.ContainsAny("password2", "MyPassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for password missing uppercase")
	}

	// Should not have error for password2
	v2 := NewValidator()
	v2.ContainsAny("password2", "MyPassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if err := v2.Validate(); err != nil {
		t.Error("Expected no error for password containing uppercase")
	}
}

func TestValidateJSON(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		jsonBody := `{"email":"test@example.com","password":"secret123"}`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, func(v *Validator, d map[string]any) *Validator {
			return v.RequiredString("email", d["email"].(string)).
				Email("email", d["email"].(string)).
				RequiredString("password", d["password"].(string)).
				MinLength("password", d["password"].(string), 8)
		})

		if data == nil {
			t.Fatal("Expected data to be returned")
		}
		if data["email"].(string) != "test@example.com" {
			t.Errorf("Expected email to be 'test@example.com', got %s", data["email"].(string))
		}
		if data["password"].(string) != "secret123" {
			t.Errorf("Expected password to be 'secret123', got %s", data["password"].(string))
		}
		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no error response, got status %d", w.Code)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		jsonBody := `{"email":"invalid","password":"short"}`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, func(v *Validator, d map[string]any) *Validator {
			return v.RequiredString("email", d["email"].(string)).
				Email("email", d["email"].(string)).
				RequiredString("password", d["password"].(string)).
				MinLength("password", d["password"].(string), 8)
		})

		if data != nil {
			t.Error("Expected data to be nil on validation error")
		}
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonBody := `{"email":"test@example.com"`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, func(v *Validator, d map[string]any) *Validator {
			return v
		})

		if data != nil {
			t.Error("Expected data to be nil on decode error")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		jsonBody := `{"email":"test@example.com"}`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, func(v *Validator, d map[string]any) *Validator {
			email, _ := d["email"].(string)
			password, _ := d["password"].(string)
			return v.RequiredString("email", email).
				Email("email", email).
				RequiredString("password", password)
		})

		if data != nil {
			t.Error("Expected data to be nil on validation error")
		}
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
		}
	})
}
