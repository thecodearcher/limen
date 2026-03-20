package sessionjwt

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/thecodearcher/limen"
)

var testJWTSecret = []byte("test-secret-key-for-jwt-signing!")

func newTestPlugin() *sessionJWTPlugin {
	return &sessionJWTPlugin{
		config: &config{
			signingMethod:        jwt.SigningMethodHS256,
			signingKey:           testJWTSecret,
			verificationKey:      testJWTSecret,
			accessTokenDuration:  15 * time.Minute,
			refreshTokenDuration: 7 * 24 * time.Hour,
			refreshTokenRotation: true,
			refreshTokenEnabled:  true,
			issuer:               "test-issuer",
			audience:             []string{"test-audience"},
			subjectEncoder:       func(user *limen.User) string { return fmt.Sprintf("%v", user.ID) },
			subjectResolver:      func(subject string) (any, error) { return subject, nil },
		},
	}
}

func TestGenerateAccessToken(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}

	signed, jti, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, signed)
	assert.NotEmpty(t, jti)
}

func TestGenerateAccessToken_ContainsClaims(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}

	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	claims, err := plugin.VerifyAccessToken(signed)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "a@b.com", claims.Email)
	assert.Equal(t, "test-issuer", claims.Issuer)
	assert.Contains(t, claims.Audience, "test-audience")
}

func TestGenerateAccessToken_CustomClaims(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	plugin.config.customClaims = func(user *limen.User) map[string]any {
		return map[string]any{"role": "admin"}
	}

	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	claims, err := plugin.VerifyAccessToken(signed)
	assert.NoError(t, err)
	assert.Equal(t, "admin", claims.Custom["role"])
}

func TestVerifyAccessToken_Expired(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	plugin.config.accessTokenDuration = -1 * time.Second // already expired

	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	_, err = plugin.VerifyAccessToken(signed)
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestVerifyAccessToken_WrongSignature(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	wrongPlugin := newTestPlugin()
	wrongPlugin.config.signingKey = []byte("completely-different-key-32bytes!")
	wrongPlugin.config.verificationKey = []byte("completely-different-key-32bytes!")

	_, err = wrongPlugin.VerifyAccessToken(signed)
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestVerifyAccessToken_InvalidToken(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()

	_, err := plugin.VerifyAccessToken("not.a.valid.token")
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestVerifyAccessToken_WrongIssuer(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, _ := plugin.GenerateAccessToken(user)

	verifier := newTestPlugin()
	verifier.config.issuer = "wrong-issuer"

	_, err := verifier.VerifyAccessToken(signed)
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestParseAccessTokenLenient_ExpiredButValid(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	plugin.config.accessTokenDuration = -1 * time.Second

	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, _ := plugin.GenerateAccessToken(user)

	claims := plugin.parseAccessTokenLenient(signed)
	assert.NotNil(t, claims)
	assert.Equal(t, "user-1", claims.Subject)
}
