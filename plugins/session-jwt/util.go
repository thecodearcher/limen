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

// encodeRefreshTokenValue combines a token and its family into the signed
// value handed to clients.
func (p *sessionJWTPlugin) encodeRefreshTokenValue(token, family string) string {
	return p.core.SignValue(token + "." + family)
}

// parseRefreshTokenValue verifies a value from encodeRefreshTokenValue and
// splits it back into its token and family.
func (p *sessionJWTPlugin) parseRefreshTokenValue(value string) (token, family string, ok bool) {
	value, ok = p.core.VerifySignedValue(value)
	if !ok {
		return "", "", false
	}
	token, family, _ = strings.Cut(value, ".")
	return token, family, true
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
		if val, ok := body["refresh_token"].(string); ok {
			raw = strings.TrimSpace(val)
		}
	}
	if raw == "" {
		return "", "", ErrMissingRefreshToken
	}
	token, family, ok := p.parseRefreshTokenValue(raw)
	if !ok {
		return "", "", ErrInvalidRefreshToken
	}
	return token, family, nil
}
