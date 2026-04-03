package twofactor

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/limen"
)

type otpHandlers struct {
	otp       *otp
	responder *limen.Responder
}

func newOTPHandlers(otp *otp, responder *limen.Responder) *otpHandlers {
	return &otpHandlers{
		otp:       otp,
		responder: responder,
	}
}

func (o *otpHandlers) SendCode(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, o.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("email", data["email"])
	})

	if body == nil {
		return
	}

	if err := o.otp.SendOTPCode(r.Context(), body["email"].(string)); err != nil && !errors.Is(err, limen.ErrRecordNotFound) {
		o.responder.Error(w, r, err)
		return
	}

	o.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "An OTP code will be sent to your email if it is associated with an account",
	})
}
