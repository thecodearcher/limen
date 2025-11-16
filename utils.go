package aegis

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateCryptoSecureRandomString generates a cryptographically secure random string
func GenerateCryptoSecureRandomString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}
