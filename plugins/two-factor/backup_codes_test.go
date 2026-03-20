package twofactor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateBackupCodes(t *testing.T) {
	t.Parallel()

	codes := generateBackupCodes(10, 10)
	assert.Len(t, codes, 10)

	for _, code := range codes {
		assert.Contains(t, code, "-", "backup code should contain a hyphen separator")
		parts := strings.Split(code, "-")
		assert.Len(t, parts, 2)
		assert.Len(t, parts[0], 5) // half of 10
		assert.Len(t, parts[1], 5)
	}
}

func TestGenerateBackupCodes_Unique(t *testing.T) {
	t.Parallel()

	codes := generateBackupCodes(20, 10)
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "backup codes should be unique, got duplicate: %s", code)
		seen[code] = true
	}
}

func TestLooksLikeBackupCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "backup code", code: "abcde-fghij", want: true},
		{name: "totp code", code: "123456", want: false},
		{name: "empty", code: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, looksLikeBackupCode(tt.code))
		})
	}
}

func TestBackupCodes_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	secret := []byte("01234567890123456789012345678901")
	plugin := &twoFactorPlugin{
		config: &config{secret: secret},
	}
	bc := newBackupCodes(plugin, NewDefaultBackupCodesConfig())

	codes := []string{"abc-def", "ghi-jkl", "mno-pqr"}
	encrypted, err := bc.encryptBackupCodes(codes)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := bc.decryptBackupCodes(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, codes, decrypted)
}

func TestBackupCodes_CheckAndExpireBackupCode(t *testing.T) {
	t.Parallel()

	secret := []byte("01234567890123456789012345678901")
	plugin := &twoFactorPlugin{
		config: &config{secret: secret},
	}
	bc := newBackupCodes(plugin, NewDefaultBackupCodesConfig())

	codes := []string{"abc-def", "ghi-jkl", "mno-pqr"}

	encrypted, valid := bc.checkAndExpireBackupCode(codes, "ghi-jkl")
	assert.True(t, valid)
	assert.NotEmpty(t, encrypted)

	remaining, err := bc.decryptBackupCodes(encrypted)
	assert.NoError(t, err)
	assert.Len(t, remaining, 2)
	assert.NotContains(t, remaining, "ghi-jkl")
}

func TestBackupCodes_CheckAndExpire_InvalidCode(t *testing.T) {
	t.Parallel()

	secret := []byte("01234567890123456789012345678901")
	plugin := &twoFactorPlugin{
		config: &config{secret: secret},
	}
	bc := newBackupCodes(plugin, NewDefaultBackupCodesConfig())

	codes := []string{"abc-def", "ghi-jkl"}

	_, valid := bc.checkAndExpireBackupCode(codes, "invalid-code")
	assert.False(t, valid)
}
