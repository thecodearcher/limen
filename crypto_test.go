package limen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plaintext      string
		additionalData []byte
	}{
		{name: "simple text", plaintext: "hello world", additionalData: nil},
		{name: "with additional data", plaintext: "secret payload", additionalData: []byte("context")},
		{name: "long text", plaintext: strings.Repeat("a", 10000), additionalData: nil},
		{name: "special characters", plaintext: `{"key": "value", "nested": {"a": 1}}`, additionalData: []byte("json")},
		{name: "single character", plaintext: "x", additionalData: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encrypted, err := EncryptXChaCha(tt.plaintext, TestSecret, tt.additionalData)
			assert.NoError(t, err)
			assert.NotEmpty(t, encrypted)
			assert.NotEqual(t, tt.plaintext, encrypted)

			decrypted, err := DecryptXChaCha(encrypted, TestSecret, tt.additionalData)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptXChaCha_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plaintext      string
		secret         []byte
		additionalData []byte
		wantErrContain string
	}{
		{
			name:           "empty plaintext",
			plaintext:      "",
			secret:         TestSecret,
			wantErrContain: "empty",
		},
		{
			name:           "short secret",
			plaintext:      "hello",
			secret:         []byte("tooshort"),
			wantErrContain: "32 bytes",
		},
		{
			name:           "nil secret",
			plaintext:      "hello",
			secret:         nil,
			wantErrContain: "32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := EncryptXChaCha(tt.plaintext, tt.secret, tt.additionalData)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrContain)
		})
	}
}

func TestDecryptXChaCha_Errors(t *testing.T) {
	t.Parallel()

	validCiphertext, _ := EncryptXChaCha("test", TestSecret, nil)
	wrongKey := []byte("99999999999999999999999999999999")

	tests := []struct {
		name           string
		ciphertext     string
		secret         []byte
		additionalData []byte
		wantErr        bool
	}{
		{name: "empty ciphertext", ciphertext: "", secret: TestSecret, wantErr: true},
		{name: "wrong key", ciphertext: validCiphertext, secret: wrongKey, wantErr: true},
		{name: "invalid base64", ciphertext: "not-base64!!!", secret: TestSecret, wantErr: true},
		{name: "tampered ciphertext", ciphertext: validCiphertext[:len(validCiphertext)-2] + "AA", secret: TestSecret, wantErr: true},
		{name: "wrong additional data", ciphertext: func() string {
			ct, _ := EncryptXChaCha("test", TestSecret, []byte("aad1"))
			return ct
		}(), secret: TestSecret, additionalData: []byte("aad2"), wantErr: true},
		{name: "short key for decrypt", ciphertext: validCiphertext, secret: []byte("short"), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := DecryptXChaCha(tt.ciphertext, tt.secret, tt.additionalData)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	t.Parallel()

	ct1, err := EncryptXChaCha("same input", TestSecret, nil)
	assert.NoError(t, err)

	ct2, err := EncryptXChaCha("same input", TestSecret, nil)
	assert.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "two encryptions of the same plaintext should produce different ciphertexts due to random nonce")
}
