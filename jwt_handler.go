package aegis

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtHandler struct {
	config      *jWTConfig
	keyProvider jWTKeyProvider
}

func newJwtHandler(config *jWTConfig) (*JwtHandler, error) {
	if !config.enabled {
		return nil, errors.New("JWT is not enabled")
	}

	keyProvider, err := newKeyProvider(&config.signing)
	if err != nil {
		return nil, fmt.Errorf("failed to create key provider: %w", err)
	}

	return &JwtHandler{
		config:      config,
		keyProvider: keyProvider,
	}, nil
}

// GenerateToken generates a JWT token with the given claims and duration
func (s *JwtHandler) GenerateToken(claims map[string]interface{}, duration time.Duration) (string, error) {
	signingMethod := s.keyProvider.getSigningMethod()
	token := jwt.New(signingMethod)

	now := time.Now()
	standardClaims := jwt.MapClaims{
		"iss": s.config.accessToken.issuer,
		"exp": now.Add(duration).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
	}

	maps.Copy(standardClaims, claims)

	token.Claims = standardClaims

	signingKey, err := s.keyProvider.getSigningKey()
	if err != nil {
		return "", fmt.Errorf("failed to get signing key: %w", err)
	}

	return token.SignedString(signingKey)
}

// VerifyToken verifies a JWT token and returns the claims
func (s *JwtHandler) VerifyToken(tokenString string) (map[string]any, error) {
	verificationKey, err := s.keyProvider.getVerificationKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get verification key: %w", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return verificationKey, nil
	}, jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{s.keyProvider.getSigningMethod().Alg()}),
		jwt.WithIssuer(s.config.accessToken.issuer))

	if err != nil {
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// GenerateAccessToken generates an access token with the configured duration and claims
// rawUserData is the raw user data as returned from the database
func (s *JwtHandler) GenerateAccessToken(user *User) (string, error) {
	rawUserData := user.Raw()
	claims := map[string]any{
		"sub": rawUserData[s.config.claims.subjectField],
	}
	for _, claim := range s.config.claims.customClaims {
		claims[claim] = rawUserData[claim]
	}
	return s.GenerateToken(claims, s.config.accessToken.duration)
}

// GenerateRefreshToken generates a refresh token with the configured duration
func (s *JwtHandler) GenerateRefreshToken(claims map[string]any) (string, error) {
	if !s.config.refreshToken.enabled {
		return "", errors.New("refresh tokens are not enabled")
	}
	return s.GenerateToken(claims, s.config.refreshToken.duration)
}
