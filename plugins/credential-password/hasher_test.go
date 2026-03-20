package credentialpassword

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestHasher() *passwordHasher {
	return newPasswordHasher(passwordHasherConfig{
		time:      1,
		memoryKiB: 16 * 1024,
		Parallel:  1,
		saltLen:   16,
		keyLen:    32,
	})
}

func TestHashPassword_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
	}{
		{name: "simple", password: "Password1"},
		{name: "long password", password: strings.Repeat("secureP1!", 20)},
		{name: "unicode", password: "Pässwörd123"},
		{name: "special chars", password: "P@$$w0rd!#%&"},
		{name: "minimum", password: "Ab1x"},
	}

	hasher := newTestHasher()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash, err := hasher.hashPassword([]byte(tt.password))
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)

			valid, err := hasher.verifyPassword([]byte(tt.password), hash)
			assert.NoError(t, err)
			assert.True(t, valid)
		})
	}
}

func TestHashPassword_WrongPassword(t *testing.T) {
	t.Parallel()

	hasher := newTestHasher()
	hash, err := hasher.hashPassword([]byte("correctPassword1"))
	assert.NoError(t, err)

	valid, err := hasher.verifyPassword([]byte("wrongPassword1"), hash)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestHashPassword_PHCFormat(t *testing.T) {
	t.Parallel()

	hasher := newTestHasher()
	hash, err := hasher.hashPassword([]byte("Test1234"))
	assert.NoError(t, err)

	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"))

	parts := strings.Split(hash, "$")
	assert.Len(t, parts, 6, "PHC format should have 6 parts separated by $")
}

func TestHashPassword_UniqueSalts(t *testing.T) {
	t.Parallel()

	hasher := newTestHasher()
	hash1, err := hasher.hashPassword([]byte("Password1"))
	assert.NoError(t, err)

	hash2, err := hasher.hashPassword([]byte("Password1"))
	assert.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "same password should produce different hashes due to random salt")
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	t.Parallel()

	hasher := newTestHasher()

	tests := []struct {
		name string
		hash string
	}{
		{name: "empty", hash: ""},
		{name: "garbage", hash: "not-a-hash"},
		{name: "wrong format", hash: "$argon2id$v=19$garbage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := hasher.verifyPassword([]byte("password"), tt.hash)
			assert.Error(t, err)
		})
	}
}
