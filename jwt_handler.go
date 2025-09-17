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
func (s *JwtHandler) GenerateAccessToken(sessionID string, user *User) (string, string, error) {
	rawUserData := user.Raw()
	claims := map[string]any{
		"jti": sessionID,
	}
	if s.config.claims.subjectValue != nil {
		claims["sub"] = s.config.claims.subjectValue(user)
	} else {
		claims["sub"] = rawUserData[s.config.claims.subjectField]
	}

	if s.config.claims.customClaims != nil {
		customClaims := s.config.claims.customClaims(user)
		maps.Copy(claims, customClaims)
	}
	accessToken, err := s.GenerateToken(claims, s.config.accessToken.duration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	if s.config.refreshToken.enabled {
		refreshClaims := claims
		refreshClaims["jti"] = sessionID + "_refresh"

		refreshToken, err := s.GenerateToken(refreshClaims, s.config.refreshToken.duration)
		if err != nil {
			return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
		}
		return accessToken, refreshToken, nil
	}

	return accessToken, "", nil
}

func (s *JwtHandler) CustomUserFromSubjectFn() func(string) (*User, error) {
	return s.config.claims.userFromSubject
}
