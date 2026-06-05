package emailotp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"math/big"
)

// otpState is the JSON-encoded value stored alongside an email-otp
// verification record. Field names are abbreviated to keep the row small.
type otpState struct {
	Email    string  `json:"e"`
	Type     OTPType `json:"t"`
	OTP      string  `json:"o"` // plaintext or HMAC-SHA256 hash, controlled by hashStoredOTP
	Attempts int     `json:"a"`
}

func encodeOTPState(s *otpState) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeOTPState(value string) (*otpState, error) {
	var s otpState
	if err := json.Unmarshal([]byte(value), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// toOTPIdentifier returns the verification subject for a given OTP type
// and email. It mirrors better-auth's `{type}-otp-{email}` convention so
// the storage layout stays self-explanatory in raw rows.
func toOTPIdentifier(t OTPType, email string) string {
	return string(t) + "-otp-" + email
}

func (p *emailOTPPlugin) generateOTP(email string, t OTPType) (string, error) {
	if p.config.generateOTP != nil {
		code, err := p.config.generateOTP(email, t)
		if err != nil {
			return "", err
		}
		if code != "" {
			return code, nil
		}
	}
	return defaultNumericOTP(p.config.otpLength)
}

// defaultNumericOTP generates a uniformly-random numeric code of n digits.
func defaultNumericOTP(n int) (string, error) {
	const digits = "0123456789"
	out := make([]byte, n)
	max := big.NewInt(int64(len(digits)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = digits[idx.Int64()]
	}
	return string(out), nil
}

// storeValue returns the value persisted in the verifications row for a given
// plaintext OTP, applying HMAC-SHA256 with the core secret when hashing is
// enabled.
func (p *emailOTPPlugin) storeValue(otp string) string {
	if !p.config.hashStoredOTP {
		return otp
	}
	return p.hashOTP(otp)
}

// matchOTP performs a constant-time comparison between a stored value and a
// freshly-presented plaintext OTP, honoring the configured storage mode.
func (p *emailOTPPlugin) matchOTP(stored, presented string) bool {
	if p.config.hashStoredOTP {
		presented = p.hashOTP(presented)
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(presented)) == 1
}

func (p *emailOTPPlugin) hashOTP(otp string) string {
	mac := hmac.New(sha256.New, p.core.Secret())
	mac.Write([]byte(otp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
