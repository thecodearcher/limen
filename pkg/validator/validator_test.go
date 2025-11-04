package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thecodearcher/aegis"
)

func TestRequired(t *testing.T) {
	v := New()
	v.Required("email", "")
	v.Required("name", "John")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for empty email")
	}

	if !strings.Contains(err.Error(), "email") {
		t.Error("Expected error to mention email field")
	}
}

func TestMinLength(t *testing.T) {
	v := New()
	v.MinLength("password", "abc", 5)
	v.MinLength("username", "johndoe", 5)

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for short password")
	}
}

func TestMaxLength(t *testing.T) {
	v := New()
	v.MaxLength("username", "thisiswaytoolong", 10)
	v.MaxLength("name", "John", 10)

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for long username")
	}
}

func TestEmail(t *testing.T) {
	v := New()
	v.Email("email", "invalid-email")
	v.Email("email2", "valid@example.com")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid email")
	}
}

func TestChaining(t *testing.T) {
	v := New()
	v.Required("email", "").
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
	v := New()
	v.In("status", "pending", []string{"active", "inactive"})
	v.In("role", "admin", []string{"admin", "user", "guest"})

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for status")
	}

	// Should not have error for role
	v2 := New()
	v2.In("role", "admin", []string{"admin", "user", "guest"})
	if err := v2.Validate(); err != nil {
		t.Error("Expected no error for valid role")
	}
}

func TestContains(t *testing.T) {
	v := New()
	v.Contains("password", "mypassword123", "!")
	v.Contains("code", "ABC123", "123")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for password missing !")
	}

	// Should not have error for code
	v2 := New()
	v2.Contains("code", "ABC123", "123")
	if err := v2.Validate(); err != nil {
		t.Error("Expected no error for code containing 123")
	}
}

func TestContainsAny(t *testing.T) {
	v := New()
	v.ContainsAny("password", "mypassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	v.ContainsAny("password2", "MyPassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	err := v.Validate()
	if err == nil {
		t.Error("Expected validation error for password missing uppercase")
	}

	// Should not have error for password2
	v2 := New()
	v2.ContainsAny("password2", "MyPassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if err := v2.Validate(); err != nil {
		t.Error("Expected no error for password containing uppercase")
	}
}

func TestDecodeJSONAndValidate(t *testing.T) {
	type TestData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	t.Run("valid data", func(t *testing.T) {
		jsonBody := `{"email":"test@example.com","password":"secret123"}`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
		w := httptest.NewRecorder()
		responder := aegis.NewResponder(nil)

		data := DecodeJSONAndValidate(w, req, &responder, func(v *Validator, d *TestData) *Validator {
			return v.Required("email", d.Email).
				Email("email", d.Email).
				Required("password", d.Password).
				MinLength("password", d.Password, 8)
		})

		if data == nil {
			t.Fatal("Expected data to be returned")
		}
		if data.Email != "test@example.com" {
			t.Errorf("Expected email to be 'test@example.com', got %s", data.Email)
		}
		if data.Password != "secret123" {
			t.Errorf("Expected password to be 'secret123', got %s", data.Password)
		}
		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no error response, got status %d", w.Code)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		jsonBody := `{"email":"invalid","password":"short"}`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
		w := httptest.NewRecorder()
		responder := aegis.NewResponder(nil)

		data := DecodeJSONAndValidate(w, req, &responder, func(v *Validator, d *TestData) *Validator {
			return v.Required("email", d.Email).
				Email("email", d.Email).
				Required("password", d.Password).
				MinLength("password", d.Password, 8)
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
		w := httptest.NewRecorder()
		responder := aegis.NewResponder(nil)

		data := DecodeJSONAndValidate(w, req, &responder, func(v *Validator, d *TestData) *Validator {
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
		w := httptest.NewRecorder()
		responder := aegis.NewResponder(nil)

		data := DecodeJSONAndValidate(w, req, &responder, func(v *Validator, d *TestData) *Validator {
			return v.Required("email", d.Email).
				Email("email", d.Email).
				Required("password", d.Password)
		})

		if data != nil {
			t.Error("Expected data to be nil on validation error")
		}
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
		}
	})
}
