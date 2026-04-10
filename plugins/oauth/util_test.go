package oauth

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeIDTokenClaims(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		wantEmail string
	}{
		{
			name:    "invalid format - no dots",
			token:   "notajwt",
			wantErr: true,
		},
		{
			name:    "invalid format - only two parts",
			token:   "header.payload",
			wantErr: true,
		},
		{
			name:    "invalid base64 payload",
			token:   "header.!!!invalid!!!.signature",
			wantErr: true,
		},
		{
			name:    "invalid JSON payload",
			token:   "header." + base64.RawURLEncoding.EncodeToString([]byte("not json")) + ".signature",
			wantErr: true,
		},
		{
			name: "valid JWT payload",
			token: buildTestJWT(t, map[string]any{
				"sub":   "123",
				"email": "user@example.com",
				"iss":   "https://accounts.google.com",
			}),
			wantEmail: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims, err := DecodeIDTokenClaims(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, claims["email"])
		})
	}
}

func TestBuildAuthCodeURL(t *testing.T) {
	t.Parallel()

	provider := &testProvider{name: "test"}
	config, _ := provider.OAuth2Config()

	tests := []struct {
		name     string
		verifier string
		wantPKCE bool
	}{
		{name: "includes PKCE when verifier provided", verifier: "verifier-value", wantPKCE: true},
		{name: "no PKCE when verifier empty", verifier: "", wantPKCE: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authURL := BuildAuthCodeURL(config, "state-token", tt.verifier)
			parsed, err := url.Parse(authURL)
			require.NoError(t, err)
			query := parsed.Query()

			assert.Equal(t, "state-token", query.Get("state"))
			if tt.wantPKCE {
				assert.Equal(t, "S256", query.Get("code_challenge_method"))
				assert.NotEmpty(t, query.Get("code_challenge"))
			} else {
				assert.Empty(t, query.Get("code_challenge"))
				assert.Empty(t, query.Get("code_challenge_method"))
			}
		})
	}
}

// buildTestJWT creates a fake JWT with the given claims payload.
// The header and signature are dummy values -- only the payload is meaningful.
func buildTestJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encodedPayload + ".fake-signature"
}
