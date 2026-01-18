package emailpassword

import (
	"context"
	"time"

	"github.com/thecodearcher/aegis"
)

type ConfigOption func(*config)

// WithPasswordMinLength sets the minimum length of the password
func WithPasswordMinLength(passwordMinLength int) ConfigOption {
	return func(c *config) {
		c.passwordMinLength = passwordMinLength
	}
}

// WithPasswordRequireUppercase sets whether to require uppercase letters in the password
func WithPasswordRequireUppercase(passwordRequireUppercase bool) ConfigOption {
	return func(c *config) {
		c.passwordRequireUppercase = passwordRequireUppercase
	}
}

// WithPasswordRequireNumbers sets whether to require numbers in the password
func WithPasswordRequireNumbers(passwordRequireNumbers bool) ConfigOption {
	return func(c *config) {
		c.passwordRequireNumbers = passwordRequireNumbers
	}
}

// WithPasswordRequireSymbols sets whether to require symbols in the password
func WithPasswordRequireSymbols(passwordRequireSymbols bool) ConfigOption {
	return func(c *config) {
		c.passwordRequireSymbols = passwordRequireSymbols
	}
}

// WithHashFn sets the function to hash the password
func WithHashFn(hashFn func(password string) (string, error)) ConfigOption {
	return func(c *config) {
		c.hashFn = hashFn
	}
}

// WithCompareFn sets the function to compare the password and the hash
func WithCompareFn(compareFn func(password string, hash string) (bool, error)) ConfigOption {
	return func(c *config) {
		c.compareFn = compareFn
	}
}

// WithPasswordHasherConfigOptions sets the Argon2id configuration for the password hasher
func WithPasswordHasherConfigOptions(opts ...PasswordHasherConfigOption) ConfigOption {
	return func(c *config) {
		c.passwordHasherConfig = DefaultPasswordHasherConfig(opts...)
	}
}

// WithResetTokenExpiration sets the expiration duration for the reset token
func WithResetTokenExpiration(resetTokenExpiration time.Duration) ConfigOption {
	return func(c *config) {
		c.resetTokenExpiration = resetTokenExpiration
	}
}

// WithGenerateResetToken sets the function to generate the reset token
func WithGenerateResetToken(generateResetToken func(*aegis.User) (string, error)) ConfigOption {
	return func(c *config) {
		c.generateResetToken = generateResetToken
	}
}

// WithAutoSignInOnSignUp sets whether to auto sign in the user after sign up
func WithAutoSignInOnSignUp(autoSignInOnSignUp bool) ConfigOption {
	return func(c *config) {
		c.autoSignInOnSignUp = autoSignInOnSignUp
	}
}

// WithSendVerificationEmail sets the function to send the email verification message
func WithSendVerificationEmail(sendVerificationEmail func(email string, token string)) ConfigOption {
	return func(c *config) {
		c.sendVerificationEmail = sendVerificationEmail
	}
}

// WithRequireEmailVerification sets whether to require email verification after sign up
func WithRequireEmailVerification(requireEmailVerification bool) ConfigOption {
	return func(c *config) {
		c.requireEmailVerification = requireEmailVerification
	}
}

// WithSendPasswordResetEmail sets the function to send the password reset message
func WithSendPasswordResetEmail(sendPasswordResetEmail func(email string, token string)) ConfigOption {
	return func(c *config) {
		c.sendPasswordResetEmail = sendPasswordResetEmail
	}
}

// WithOnPasswordResetSuccess sets the function to call when the password reset is successful
func WithOnPasswordResetSuccess(onPasswordResetSuccess func(ctx context.Context, user *aegis.User)) ConfigOption {
	return func(c *config) {
		c.onPasswordResetSuccess = onPasswordResetSuccess
	}
}

// WithEmailVerificationExpiration sets the expiration duration for the email verification
func WithEmailVerificationExpiration(emailVerificationExpiration time.Duration) ConfigOption {
	return func(c *config) {
		c.emailVerificationExpiration = emailVerificationExpiration
	}
}
