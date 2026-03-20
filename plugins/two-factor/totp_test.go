package twofactor

import (
	"testing"

	"encoding/base32"

	"github.com/stretchr/testify/assert"

	"github.com/thecodearcher/limen"
)

func newTestTOTP() *totp {
	return &totp{
		totpConfig: NewDefaultTOTPConfig(WithTOTPIssuer("TestApp")),
	}
}

func TestTOTP_GenerateSetupURI(t *testing.T) {
	t.Parallel()

	totp := newTestTOTP()

	uri, err := totp.GenerateSetupURI("user@example.com", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, uri.URI)
	assert.NotEmpty(t, uri.Secret)
	assert.Contains(t, uri.URI, "otpauth://totp/")
	assert.Contains(t, uri.URI, "TestApp")
	assert.Contains(t, uri.URI, "user@example.com")
}

func TestTOTP_GenerateSetupURI_WithExistingSecret(t *testing.T) {
	t.Parallel()

	totpInstance := newTestTOTP()

	secret := limen.GenerateRandomString(32, limen.CharSetAlphanumeric)

	uri, err := totpInstance.GenerateSetupURI("a@b.com", secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, uri.URI)
	assert.NotEmpty(t, uri.Secret, "should produce a non-empty secret")
	assert.Contains(t, uri.URI, "a@b.com")
	b32NoPadding, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(uri.Secret)
	assert.Equal(t, secret, string(b32NoPadding))
}
