package twofactor

import (
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/validator"
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
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		o.responder.Error(w, r, err)
		return
	}

	user := o.otp.plugin.userSchema.UserToUserWithTwoFactor(session.User)

	err = o.otp.SendOTPCode(r.Context(), user)
	if err != nil {
		o.responder.Error(w, r, err)
		return
	}

	o.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "OTP code sent successfully",
	})
}

func (o *otpHandlers) VerifyCode(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, o.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
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

	user := o.otp.plugin.userSchema.UserToUserWithTwoFactor(session.User)
	err = o.otp.Verify(r.Context(), user, body["code"].(string))
	if err != nil {
		o.responder.Error(w, r, err)
		return
	}

	o.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "OTP code verified successfully",
	})
}
