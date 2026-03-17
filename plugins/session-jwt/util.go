package sessionjwt

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/thecodearcher/limen"
)

func generateOpaqueToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// encodeRefreshTokenValue combines a token and its family into a single
// dot-delimited string for inclusion in response bodies.
func encodeRefreshTokenValue(token, family string) string {
	return token + "." + family
}

// parseRefreshTokenValue splits an encoded token.family value.
func parseRefreshTokenValue(raw string) (token, family string) {
	token, family, _ = strings.Cut(raw, ".")
	return token, family
}

func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (p *sessionJWTPlugin) extractToken(r *http.Request) (string, error) {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				return token, nil
			}
		}
	}

	return "", limen.ErrSessionNotFound
}

// extractRefreshToken reads the refresh token from the JSON request body
// and parses the token.family encoded value.
func (p *sessionJWTPlugin) extractRefreshToken(r *http.Request) (token string, family string, err error) {
	var raw string
	if body := limen.GetJSONBody(r); body != nil {
		if val, ok := body["refreshToken"].(string); ok {
			raw = strings.TrimSpace(val)
		}
	}
	if raw == "" {
		return "", "", ErrMissingRefreshToken
	}
	token, family = parseRefreshTokenValue(raw)
	return token, family, nil
}
