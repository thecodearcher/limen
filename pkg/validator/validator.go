package validator

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/httpx"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s %s", e.Field, e.Message)
	}
	return e.Message
}

type Errors struct {
	errors []*ValidationError
}

func (e *Errors) Error() string {
	if len(e.errors) == 0 {
		return ""
	}
	if len(e.errors) == 1 {
		return e.errors[0].Error()
	}
	messages := make([]string, len(e.errors))
	for i, err := range e.errors {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

func (e *Errors) Add(field, message string) {
	e.errors = append(e.errors, &ValidationError{
		Field:   field,
		Message: message,
	})
}

func (e *Errors) HasErrors() bool {
	return len(e.errors) > 0
}

func (e *Errors) GetErrors() []*ValidationError {
	return e.errors
}

type Validator struct {
	errors *Errors
}

func New() *Validator {
	return &Validator{
		errors: &Errors{},
	}
}

func (v *Validator) Validate() error {
	if v.errors.HasErrors() {
		return v.errors
	}
	return nil
}

func (v *Validator) Required(field string, value any) *Validator {
	if value == nil {
		v.errors.Add(field, "is required")
		return v
	}

	valueString, ok := value.(string)
	if ok && strings.TrimSpace(valueString) == "" {
		v.errors.Add(field, "is required")
	}

	return v
}

func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.errors.Add(field, fmt.Sprintf("must be at least %d characters", min))
	}
	return v
}

func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.errors.Add(field, fmt.Sprintf("must be at most %d characters", max))
	}
	return v
}

func (v *Validator) Length(field, value string, length int) *Validator {
	if len(value) != length {
		v.errors.Add(field, fmt.Sprintf("must be exactly %d characters", length))
	}
	return v
}

func (v *Validator) Email(field string, value any) *Validator {
	if value == nil || value == "" {
		return v
	}
	emailRegex := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailRegex, value.(string))
	if err != nil || !matched {
		v.errors.Add(field, "must be a valid email address")
	}
	return v
}

func (v *Validator) Custom(field string, valid bool, message string) *Validator {
	if !valid {
		v.errors.Add(field, message)
	}
	return v
}

func (v *Validator) URL(field, value string) *Validator {
	if value == "" {
		return v // Empty URLs are handled by Required()
	}
	urlRegex := `^https?://[^\s/$.?#].[^\s]*$`
	matched, err := regexp.MatchString(urlRegex, value)
	if err != nil || !matched {
		v.errors.Add(field, "must be a valid URL")
	}
	return v
}

func (v *Validator) In(field, value string, allowed []string) *Validator {
	if slices.Contains(allowed, value) {
		return v
	}
	v.errors.Add(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
	return v
}

func (v *Validator) Contains(field, value, substr string) *Validator {
	if !strings.Contains(value, substr) {
		v.errors.Add(field, fmt.Sprintf("must contain '%s'", substr))
	}
	return v
}

func (v *Validator) ContainsAny(field, value, chars string) *Validator {
	if !strings.ContainsAny(value, chars) {
		v.errors.Add(field, fmt.Sprintf("must contain at least one of: %s", chars))
	}
	return v
}

func (v *Validator) NotContains(field, value, substr string) *Validator {
	if strings.Contains(value, substr) {
		v.errors.Add(field, fmt.Sprintf("must not contain '%s'", substr))
	}
	return v
}

func (v *Validator) Matches(field, value, pattern string) *Validator {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		v.errors.Add(field, "invalid pattern")
		return v
	}
	if !matched {
		v.errors.Add(field, "does not match required format")
	}
	return v
}

type DecodeError struct {
	Message string
}

func (e *DecodeError) Error() string {
	return e.Message
}

// ValidateJSON decodes the JSON body of the request and validates it using the validateFunc.
// It returns the decoded data if the validation succeeds, otherwise it returns nil and an error is written to the response.
func ValidateJSON(w http.ResponseWriter, r *http.Request, responder *aegis.Responder, validateFunc func(*Validator, map[string]any) *Validator) map[string]any {
	body := httpx.GetJSONBody(r)

	if len(body) == 0 || body == nil {
		responder.Error(w, r, aegis.NewAegisError("empty JSON body", http.StatusBadRequest, nil))
		return nil
	}

	v := New()
	validateFunc(v, body)

	if err := v.Validate(); err != nil {
		responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusUnprocessableEntity, nil))
		return nil
	}

	return body
}
