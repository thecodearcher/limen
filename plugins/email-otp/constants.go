package emailotp

// EmailOTPAction is the verification action name used for all
// email-OTP verifications persisted in the verifications table.
const EmailOTPAction = "email_otp"

// OTPType discriminates the purpose of an OTP within the email-otp plugin.
// It allows a single verification table to safely hold concurrent OTPs for
// different flows (e.g. sign-in and email-verification for the same address).
type OTPType string

const (
	// TypeSignIn is the OTP type used for the sign-in flow.
	TypeSignIn OTPType = "sign-in"
	// TypeEmailVerification is the OTP type used to verify an existing
	// user's email address.
	TypeEmailVerification OTPType = "email-verification"
)

func (t OTPType) valid() bool {
	switch t {
	case TypeSignIn, TypeEmailVerification:
		return true
	}
	return false
}
