package emailotp

import (
	"net/http"
	"strings"

	"github.com/thecodearcher/limen"
)

type emailOTPHandlers struct {
	plugin    *emailOTPPlugin
	httpCore  *limen.LimenHTTPCore
	responder *limen.Responder
}

func newEmailOTPHandlers(plugin *emailOTPPlugin, httpCore *limen.LimenHTTPCore) *emailOTPHandlers {
	return &emailOTPHandlers{
		plugin:    plugin,
		httpCore:  httpCore,
		responder: httpCore.Responder,
	}
}

func (h *emailOTPHandlers) SendOTP(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("email", data["email"]).Email("email", data["email"])
	})
	if body == nil {
		return
	}

	otpType := TypeSignIn
	if raw, ok := body["type"].(string); ok && raw != "" {
		otpType = OTPType(strings.ToLower(raw))
		if !otpType.valid() {
			h.responder.Error(w, r, ErrInvalidOTPType)
			return
		}
	}

	if _, err := h.plugin.SendOTP(r.Context(), body["email"].(string), &SendOTPOptions{Type: otpType}); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, map[string]any{
		"success": true,
		"message": "If the email exists, an OTP has been sent",
	})
}

func (h *emailOTPHandlers) SignIn(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.
			RequiredString("email", data["email"]).
			Email("email", data["email"]).
			RequiredString("otp", data["otp"])
	})
	if body == nil {
		return
	}

	additional := extractAdditionalData(body)

	result, err := h.plugin.SignInWithOTP(r.Context(), body["email"].(string), body["otp"].(string), &SignInOptions{AdditionalData: additional})
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessionResult, err := h.plugin.core.CreateSession(r.Context(), r, w, result)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.SessionResponse(w, r, h.plugin.core, result, sessionResult)
}

func (h *emailOTPHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.
			RequiredString("email", data["email"]).
			Email("email", data["email"]).
			RequiredString("otp", data["otp"])
	})
	if body == nil {
		return
	}

	result, err := h.plugin.VerifyOTP(r.Context(), body["email"].(string), body["otp"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, map[string]any{
		"success": true,
		"user":    limen.SerializeModel(h.plugin.core.Schema.User, result.User),
	})
}

// extractAdditionalData copies non-reserved keys from the sign-in request body
// so first_name, last_name, etc. flow into the auto-created user.
func extractAdditionalData(body map[string]any) map[string]any {
	reserved := map[string]struct{}{
		"email": {},
		"otp":   {},
		"type":  {},
	}
	out := make(map[string]any, len(body))
	for k, v := range body {
		if _, skip := reserved[k]; skip {
			continue
		}
		out[k] = v
	}
	return out
}
