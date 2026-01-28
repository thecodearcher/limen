package aegis

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// Encrypt encrypts the plaintext using XChaCha20-Poly1305 with the provided key (32 bytes).
func EncryptXChaCha(plaintext string, secret []byte, additionalData []byte) (string, error) {
	if len(secret) < chacha20poly1305.KeySize {
		return "", fmt.Errorf("secret key must be %d bytes", chacha20poly1305.KeySize)
	}

	xChacha, err := chacha20poly1305.NewX(secret)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, xChacha.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := xChacha.Seal(nonce, nonce, []byte(plaintext), additionalData)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the base64-encoded ciphertext using XChaCha20-Poly1305 with the provided key.
func DecryptXChaCha(encoded string, secret []byte, additionalData []byte) (string, error) {
	if len(secret) != chacha20poly1305.KeySize {
		return "", fmt.Errorf("secret key must be %d bytes", chacha20poly1305.KeySize)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	xChacha, err := chacha20poly1305.NewX(secret)
	if err != nil {
		return "", err
	}

	nonceSize := xChacha.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := xChacha.Open(nil, nonce, encrypted, additionalData)
	return string(plaintext), err
}
