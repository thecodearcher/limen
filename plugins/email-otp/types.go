package emailotp

import "time"

type config struct {
	otpExpiration   time.Duration
	otpLength       int
	allowedAttempts int
	disableSignUp   bool
	hashStoredOTP   bool
	generateOTP     func(email string, otpType OTPType) (string, error)
	sendOTP         func(EmailOTPMessage)
}

// ConfigOption configures the email-otp plugin.
type ConfigOption func(*config)

// WithOTPExpiration sets how long a generated OTP remains valid. Defaults to 5 minutes.
func WithOTPExpiration(d time.Duration) ConfigOption {
	return func(c *config) { c.otpExpiration = d }
}

// WithOTPLength sets the number of digits in a generated OTP. Defaults to 6.
// Ignored when WithGenerateOTP supplies a custom generator.
func WithOTPLength(n int) ConfigOption {
	return func(c *config) { c.otpLength = n }
}

// WithAllowedAttempts sets the number of failed verifications tolerated before
// the OTP is invalidated. Defaults to 3.
func WithAllowedAttempts(n int) ConfigOption {
	return func(c *config) { c.allowedAttempts = n }
}

// WithDisableSignUp prevents the sign-in flow from auto-creating a user when
// no account exists for the provided email. Defaults to false.
func WithDisableSignUp(disable bool) ConfigOption {
	return func(c *config) { c.disableSignUp = disable }
}

// WithHashStoredOTP enables HMAC-SHA256 hashing of the OTP at rest. With this
// enabled a stolen database snapshot does not yield usable OTPs without the
// Limen secret. Defaults to false (matching the better-auth default), and
// recommended for production.
func WithHashStoredOTP(hash bool) ConfigOption {
	return func(c *config) { c.hashStoredOTP = hash }
}

// WithGenerateOTP overrides the default numeric OTP generator. Returning an
// empty string falls back to the default generator.
func WithGenerateOTP(fn func(email string, otpType OTPType) (string, error)) ConfigOption {
	return func(c *config) { c.generateOTP = fn }
}

// WithSendOTP sets the callback used to deliver an OTP to the user. Wire your
// transactional email provider here; this callback is required for any real
// deployment.
func WithSendOTP(fn func(EmailOTPMessage)) ConfigOption {
	return func(c *config) { c.sendOTP = fn }
}

// EmailOTPMessage is the payload handed to the WithSendOTP callback for delivery.
type EmailOTPMessage struct {
	Email string
	OTP   string
	Type  OTPType
}

// SendOTPOptions carries optional parameters for SendOTP.
type SendOTPOptions struct {
	// Type defaults to TypeSignIn when zero.
	Type OTPType
}

// SignInOptions carries optional parameters for SignInWithOTP, including
// user fields used only when auto-creating a new account.
type SignInOptions struct {
	AdditionalData map[string]any
}
