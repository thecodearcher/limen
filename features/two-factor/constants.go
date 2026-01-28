package twofactor

import "github.com/thecodearcher/aegis"

const (
	TwoFactorSchemaTableName aegis.SchemaTableName = "two_factors"

	TwoFactorSchemaUserIDField      aegis.SchemaField = "user_id"
	TwoFactorSchemaSecretField      aegis.SchemaField = "secret"
	TwoFactorSchemaBackupCodesField aegis.SchemaField = "backup_codes"

	UserWithTwoFactorSchemaEnabledField aegis.SchemaField = "two_factor_enabled"
)

const (
	otpAction = "two_factor_otp"
)

// TOTPAlgorithm represents the hashing function to use in the HMAC
// operation needed for TOTP.
type TOTPAlgorithm int

const (
	// TOTPAlgorithmSHA1 should be used for compatibility with Google Authenticator.
	TOTPAlgorithmSHA1 TOTPAlgorithm = iota
	TOTPAlgorithmSHA256
	TOTPAlgorithmSHA512
	TOTPAlgorithmMD5
)

// TOTPDigits represents the number of digits present in the
// user's OTP passcode. Six and Eight are the most common values.
type TOTPDigits int

const (
	TOTPDigitsSix   TOTPDigits = 6
	TOTPDigitsEight TOTPDigits = 8
)
