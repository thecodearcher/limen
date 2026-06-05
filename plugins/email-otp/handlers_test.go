package emailotp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendOTPHandler_ValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedSubstr string
	}{
		{
			name:           "missing email field",
			body:           `{"type":"sign-in"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedSubstr: "email",
		},
		{
			name:           "invalid email",
			body:           `{"email":"not-an-email"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedSubstr: "email",
		},
		{
			name:           "invalid type",
			body:           `{"email":"u@test.com","type":"shenanigans"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedSubstr: "otp type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, _ := newTestLimenAndPlugin(t)

			req := newJSONRequest(t, http.MethodPost, "/auth/email-otp/send-otp", tt.body)
			w := httptest.NewRecorder()
			l.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedSubstr)
		})
	}
}

func TestSendOTPHandler_AlwaysReturnsGenericSuccess(t *testing.T) {
	t.Parallel()

	// Even when the user does not exist for email-verification, the handler
	// must return 200 to prevent enumeration.
	l, _ := newTestLimenAndPlugin(t, WithSendOTP(func(EmailOTPMessage) {}))

	req := newJSONRequest(t, http.MethodPost, "/auth/email-otp/send-otp", `{"email":"ghost@test.com","type":"email-verification"}`)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestSignInHandler_HappyPath_SetsSessionCookie(t *testing.T) {
	t.Parallel()

	var otp string
	l, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "handler@test.com")
	require.NoError(t, err)
	require.NotEmpty(t, otp)

	body, _ := json.Marshal(map[string]any{"email": "handler@test.com", "otp": otp})
	req := newJSONRequest(t, http.MethodPost, "/auth/email-otp/sign-in", string(body))
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotEmpty(t, w.Result().Cookies(), "successful sign-in must set a session cookie")
}

func TestSignInHandler_WrongOTPReturns400(t *testing.T) {
	t.Parallel()

	var otp string
	l, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "wrong@test.com")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{"email": "wrong@test.com", "otp": wrongOTP(otp)})
	req := newJSONRequest(t, http.MethodPost, "/auth/email-otp/sign-in", string(body))
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid otp")
	assert.Empty(t, w.Result().Cookies(), "failed sign-in must not set a session cookie")
}

func TestVerifyEmailHandler_HappyPath(t *testing.T) {
	t.Parallel()

	var otp string
	l, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	// Seed user and request email-verification OTP.
	// Using the in-test helper to bypass sign-in.
	seed := struct{ email string }{"verify-h@test.com"}
	require.NotNil(t, l)
	// SeedTestUser is used elsewhere; for handler tests we just create via sign-in first.
	_, err := plugin.SendOTP(context.Background(), seed.email)
	require.NoError(t, err)
	_, err = plugin.SignInWithOTP(context.Background(), seed.email, otp)
	require.NoError(t, err)

	// Now request an email-verification OTP and verify via handler.
	_, err = plugin.SendOTP(context.Background(), seed.email, &SendOTPOptions{Type: TypeEmailVerification})
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{"email": seed.email, "otp": otp})
	req := newJSONRequest(t, http.MethodPost, "/auth/email-otp/verify-email", string(body))
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "success")
}
