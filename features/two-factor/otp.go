package twofactor

import (
	"context"
	"errors"

	"github.com/thecodearcher/aegis"
)

type otp struct {
	*otpConfig
	plugin *twoFactorFeature
}

func newDefaultOTP(plugin *twoFactorFeature, config *otpConfig) *otp {
	return &otp{
		otpConfig: config,
		plugin:    plugin,
	}
}

func (o *otp) registerRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	handlers := newOTPHandlers(o, httpCore.Responder)
	routeBuilder.ProtectedPOST("/otp/send", "otp-send", handlers.SendCode)
	routeBuilder.ProtectedPOST("/otp/verify", "otp-verify", handlers.VerifyCode)
}

// SendOTPCode generates and sends an OTP code to the user
func (o *otp) SendOTPCode(ctx context.Context, user *UserWithTwoFactor) error {
	code := generateRandomOTP(o.digits)

	_, err := o.plugin.core.DBAction.CreateVerification(
		ctx,
		otpAction,
		user.Email,
		code,
		o.ttl,
	)
	if err != nil {
		return err
	}

	if o.sendCode != nil {
		o.sendCode(ctx, user, code)
	}

	return nil
}

func (o *otp) Verify(ctx context.Context, user *UserWithTwoFactor, code string) error {
	if err := o.plugin.core.DBAction.VerifyVerificationToken(ctx, code, otpAction, user.Email); err != nil {
		if errors.Is(err, aegis.ErrVerificationTokenInvalid) {
			return ErrInvalidOTPCode
		}
		return err
	}
	return nil
}
