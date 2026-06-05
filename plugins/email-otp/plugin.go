// Package emailotp provides one-time-code email authentication for Limen.
//
// The plugin supports two flows out of the box:
//   - sign-in (POST /email-otp/send-otp + POST /sign-in/email-otp): a
//     passwordless login that auto-creates the user when missing.
//   - email-verification (POST /email-otp/send-otp + POST /email-otp/verify-email):
//     marks an existing user's email as verified.
//
// The plugin is modelled after better-auth's email-otp plugin and reuses
// Limen's verifications table for storage. See plugin docs for configuration
// options.
package emailotp

import (
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

const (
	defaultOTPLength       = 6
	defaultOTPExpiration   = 5 * time.Minute
	defaultAllowedAttempts = 3
)

type emailOTPPlugin struct {
	core               *limen.LimenCore
	config             *config
	userSchema         *limen.UserSchema
	verificationSchema *limen.VerificationSchema
	dbAction           *limen.DatabaseActionHelper
}

// New returns an email-otp plugin configured with defaults that match
// better-auth's behavior: 6-digit numeric codes, 5-minute expiry, and 3
// allowed verification attempts before the code is invalidated.
func New(opts ...ConfigOption) *emailOTPPlugin {
	cfg := &config{
		otpExpiration:   defaultOTPExpiration,
		otpLength:       defaultOTPLength,
		allowedAttempts: defaultAllowedAttempts,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &emailOTPPlugin{config: cfg}
}

func (p *emailOTPPlugin) Name() limen.PluginName {
	return limen.PluginEmailOTP
}

func (p *emailOTPPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core
	p.userSchema = core.Schema.User
	p.verificationSchema = core.Schema.Verification
	p.dbAction = core.DBAction

	if p.config == nil {
		return fmt.Errorf("email-otp: config is required")
	}
	if p.config.otpExpiration <= 0 {
		return fmt.Errorf("email-otp: otp expiration must be positive")
	}
	if p.config.otpLength <= 0 {
		return fmt.Errorf("email-otp: otp length must be positive")
	}
	if p.config.allowedAttempts <= 0 {
		return fmt.Errorf("email-otp: allowed attempts must be positive")
	}
	return nil
}

func (p *emailOTPPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/email-otp",
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("/send-otp", 3, time.Minute),
			limen.NewRateLimitRule("/sign-in", 3, time.Minute),
			limen.NewRateLimitRule("/verify-email", 3, time.Minute),
		},
	}
}

func (p *emailOTPPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newEmailOTPHandlers(p, httpCore)
	routeBuilder.POST("/send-otp", "email-otp-send", handlers.SendOTP)
	routeBuilder.POST("/sign-in", "email-otp-sign-in", handlers.SignIn)
	routeBuilder.POST("/verify-email", "email-otp-verify-email", handlers.VerifyEmail)
}
