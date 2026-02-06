package twofactor

import (
	"context"
	"fmt"
	"time"

	pqotp "github.com/pquerna/otp"
	pqtotp "github.com/pquerna/otp/totp"

	"github.com/thecodearcher/aegis"
)

type totp struct {
	*totpConfig
	plugin *twoFactorFeature
}

type TwoFactorSetupURI struct {
	URI    string `json:"uri"`
	Secret string `json:"-"`
}

func newDefaultTOTP(plugin *twoFactorFeature, config *totpConfig) *totp {
	t := &totp{
		totpConfig: config,
		plugin:     plugin,
	}

	return t
}

// RegisterRoutes registers TOTP-specific routes
func (t *totp) registerRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	handlers := newTOTPHandlers(t, httpCore.Responder)
	routeBuilder.ProtectedGET("/totp/uri", "totp-uri", handlers.GetSetupURI)
	routeBuilder.ProtectedPOST("/totp/verify", "totp-verify", handlers.VerifyCode)
}

func (t *totp) GetSetupURI(ctx context.Context, user *UserWithTwoFactor) (*TwoFactorSetupURI, error) {
	twoFactor, err := t.plugin.FindTwoFactorByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	decryptedSecret, err := t.plugin.decrypt(twoFactor.Secret)
	if err != nil {
		return nil, err
	}

	return t.GenerateSetupURI(user.Email, decryptedSecret)
}

// GenerateSetupURI generates a URL for QR code generation
func (t *totp) GenerateSetupURI(email string, secret string) (*TwoFactorSetupURI, error) {
	opts := pqtotp.GenerateOpts{
		Issuer:      t.issuer,
		AccountName: email,
		Period:      uint(t.ttl.Seconds()),
		Digits:      pqotp.Digits(t.digits),
		Algorithm:   pqotp.Algorithm(t.algorithm),
	}

	if secret != "" {
		opts.Secret = []byte(secret)
	}

	key, err := pqtotp.Generate(opts)

	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	return &TwoFactorSetupURI{
		URI:    key.URL(),
		Secret: key.Secret(),
	}, nil
}

// VerifyCode validates a TOTP code
func (t *totp) VerifyCode(ctx context.Context, userID any, code string) error {
	twoFactor, err := t.plugin.FindTwoFactorByUserID(ctx, userID)
	if err != nil {
		return err
	}
	decryptedSecret, err := t.plugin.decrypt(twoFactor.Secret)
	if err != nil {
		return err
	}

	valid, err := pqtotp.ValidateCustom(code, decryptedSecret, time.Now().UTC(), pqtotp.ValidateOpts{
		Period:    uint(t.ttl.Seconds()),
		Digits:    pqotp.Digits(t.digits),
		Algorithm: pqotp.Algorithm(t.algorithm),
		Skew:      1,
	})

	if err != nil {
		return ErrInvalidCode
	}

	if !valid {
		return ErrInvalidCode
	}

	return nil
}
