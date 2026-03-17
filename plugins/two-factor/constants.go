package twofactor

import (
	"time"

	"github.com/thecodearcher/limen"
)

const (
	TwoFactorSchemaTableName limen.SchemaTableName = "two_factors"

	TwoFactorSchemaUserIDField      limen.SchemaField = "user_id"
	TwoFactorSchemaSecretField      limen.SchemaField = "secret"
	TwoFactorSchemaBackupCodesField limen.SchemaField = "backup_codes"

	UserWithTwoFactorSchemaEnabledField limen.SchemaField = "two_factor_enabled"
)

const (
	otpAction = "two_factor_otp"
)

const (
	defaultChallengeCookieName = "session_2fa"
	defaultChallengeExpiration = 5 * time.Minute
	challengeTokenType         = "2fa_challenge"
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

type TwoFactorMethod string

const (
	TwoFactorMethodOTP  TwoFactorMethod = "otp"
	TwoFactorMethodTOTP TwoFactorMethod = "totp"
)
