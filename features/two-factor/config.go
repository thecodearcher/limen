package twofactor

import (
	"context"
	"time"
)

type ConfigOption func(*config)

type config struct {
	secret           []byte
	totp             *totpConfig
	otp              *otpConfig
	backupCodes      *backupCodesConfig
	cookieExpiration time.Duration
	cookieName       string
}

func WithSecret(secret []byte) ConfigOption {
	return func(c *config) {
		c.secret = secret
	}
}

func WithTOTP(totp ...TOTPOption) ConfigOption {
	return func(c *config) {
		c.totp = NewDefaultTOTPConfig(totp...)
	}
}

func WithOTP(otp ...OTPOption) ConfigOption {
	return func(c *config) {
		c.otp = NewDefaultOTPConfig(otp...)
	}
}

func WithBackupCodes(backupCodes ...BackupCodesOption) ConfigOption {
	return func(c *config) {
		c.backupCodes = NewDefaultBackupCodesConfig(backupCodes...)
	}
}

type TOTPOption func(*totpConfig)

type totpConfig struct {
	issuer    string
	ttl       time.Duration
	digits    TOTPDigits
	algorithm TOTPAlgorithm
}

func NewDefaultTOTPConfig(opts ...TOTPOption) *totpConfig {
	config := &totpConfig{
		issuer:    "Aegis",
		ttl:       30 * time.Second,
		digits:    TOTPDigitsSix,
		algorithm: TOTPAlgorithmSHA1,
	}
	for _, opt := range opts {
		opt(config)
	}

	return config
}

func WithTOTPIssuer(issuer string) TOTPOption {
	return func(c *totpConfig) {
		c.issuer = issuer
	}
}

func WithTOTPTTL(ttl time.Duration) TOTPOption {
	return func(c *totpConfig) {
		c.ttl = ttl
	}
}

func WithTOTPDigits(digits TOTPDigits) TOTPOption {
	return func(c *totpConfig) {
		c.digits = digits
	}
}

func WithTOTPAlgorithm(algorithm TOTPAlgorithm) TOTPOption {
	return func(c *totpConfig) {
		c.algorithm = algorithm
	}
}

type OTPOption func(*otpConfig)

type otpConfig struct {
	enabled  bool
	ttl      time.Duration
	sendCode func(ctx context.Context, user *UserWithTwoFactor, code string)
	digits   TOTPDigits
}

func NewDefaultOTPConfig(opts ...OTPOption) *otpConfig {
	config := &otpConfig{
		enabled: true,
		ttl:     10 * time.Minute,
		digits:  TOTPDigitsSix,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

func WithOTPEnabled(enabled bool) OTPOption {
	return func(c *otpConfig) {
		c.enabled = enabled
	}
}

func WithOTPCodeExpiration(expiration time.Duration) OTPOption {
	return func(c *otpConfig) {
		c.ttl = expiration
	}
}

func WithOTPSendCode(sendCode func(ctx context.Context, user *UserWithTwoFactor, code string)) OTPOption {
	return func(c *otpConfig) {
		c.sendCode = sendCode
	}
}

func WithOTPDigits(digits TOTPDigits) OTPOption {
	return func(c *otpConfig) {
		c.digits = digits
	}
}

type backupCodesConfig struct {
	count           int
	length          int
	customGenerator func() []string
}

func NewDefaultBackupCodesConfig(opts ...BackupCodesOption) *backupCodesConfig {
	config := &backupCodesConfig{
		count:  10,
		length: 10,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

type BackupCodesOption func(*backupCodesConfig)

func WithBackupCodesCount(count int) BackupCodesOption {
	return func(c *backupCodesConfig) {
		c.count = count
	}
}

func WithBackupCodesLength(length int) BackupCodesOption {
	return func(c *backupCodesConfig) {
		c.length = length
	}
}

func WithBackupCodesCustomGenerator(generator func() []string) BackupCodesOption {
	return func(c *backupCodesConfig) {
		c.customGenerator = generator
	}
}

func WithCookieExpiration(expiration time.Duration) ConfigOption {
	return func(c *config) {
		c.cookieExpiration = expiration
	}
}

func WithCookieName(name string) ConfigOption {
	return func(c *config) {
		c.cookieName = name
	}
}
