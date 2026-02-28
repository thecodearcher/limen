package twofactor

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/aegis"
)

type otpHandlers struct {
	otp       *otp
	responder *aegis.Responder
}

func newOTPHandlers(otp *otp, responder *aegis.Responder) *otpHandlers {
	return &otpHandlers{
		otp:       otp,
		responder: responder,
	}
}

func (o *otpHandlers) SendCode(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, o.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("email", data["email"])
	})

	if body == nil {
		return
	}

	if err := o.otp.SendOTPCode(r.Context(), body["email"].(string)); err != nil && !errors.Is(err, aegis.ErrRecordNotFound) {
		o.responder.Error(w, r, err)
		return
	}

	o.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "An OTP code will be sent to your email if it is associated with an account",
	})
}

func (o *otpHandlers) VerifyCode(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, o.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("code", data["code"])
	})

	if body == nil {
		return
	}

	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		o.responder.Error(w, r, err)
		return
	}

	err = o.otp.Verify(r.Context(), session.User.ID, body["code"].(string))
	if err != nil {
		o.responder.Error(w, r, err)
		return
	}

	o.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "OTP code verified successfully",
	})
}
