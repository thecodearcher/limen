package aegis

import (
	"fmt"
	"os"
	"time"
)

// JWTConfig defines JWT-specific configuration
type jWTConfig struct {
	enabled      bool
	accessToken  accessTokenConfig
	refreshToken refreshTokenConfig
	signing      signingConfig
	claims       claimsConfig
}

// accessTokenConfig configures access token properties
type accessTokenConfig struct {
	duration time.Duration
	issuer   string
}

// refreshTokenConfig configures refresh token properties
type refreshTokenConfig struct {
	enabled  bool
	duration time.Duration
	rotate   bool // Generate new refresh token on use
}

// signingConfig configures token signing
type signingConfig struct {
	algorithm JWTAlgorithm // HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512

	// HMAC configuration
	secret string

	// RSA/ECDSA configuration
	privateKeyPath string
	publicKeyPath  string
	privateKeyPEM  string
	publicKeyPEM   string
}

// claimsConfig configures JWT claims
type claimsConfig struct {
	subjectField    string
	subjectValue    func(user *User) string
	userFromSubject func(subject string) (*User, error)
	customClaims    func(user *User) map[string]any // map of custom claims
}

// NewDefaultJWTConfig creates a new JWT configuration with default values
func NewDefaultJWTConfig(opts ...jWTConfigOption) *jWTConfig {
	config := &jWTConfig{
		enabled: true,
		accessToken: accessTokenConfig{
			duration: 60 * time.Minute,
			issuer:   "aegis",
		},
		refreshToken: refreshTokenConfig{
			enabled:  true,
			duration: 7 * 24 * time.Hour,
			rotate:   true,
		},
		signing: signingConfig{
			algorithm: "HS256",
			secret:    os.Getenv("JWT_SECRET"),
		},
		claims: claimsConfig{
			subjectField: "id",
		},
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Validate validates the JWT configuration
func (c *jWTConfig) validate() error {
	if !c.enabled {
		return nil
	}

	if c.signing.algorithm == "" {
		return ErrJWTInvalidAlgorithm
	}

	if !isHMACAlgorithm(c.signing.algorithm) && !isAsymmetricAlgorithm(c.signing.algorithm) {
		return ErrJWTInvalidAlgorithm
	}

	if isHMACAlgorithm(c.signing.algorithm) && c.signing.secret == "" {
		return ErrJWTMissingSecret
	}

	if err := c.validateAsymmetricAlgorithm(); err != nil {
		return err
	}

	if c.accessToken.duration <= 0 {
		return ErrJWTInvalidDuration
	}

	if c.refreshToken.enabled && c.refreshToken.duration <= 0 {
		return ErrJWTInvalidDuration
	}

	if c.refreshToken.enabled && c.refreshToken.duration <= c.accessToken.duration {
		return ErrJWTInvalidRefreshDuration
	}

	if c.accessToken.issuer == "" {
		return ErrJWTInvalidIssuer
	}

	if c.claims.subjectValue != nil && (c.claims.subjectField != "" && c.claims.subjectField != "id") {
		return ErrJWTSubjectFieldConflict
	}

	return nil
}

func (c *jWTConfig) validateAsymmetricAlgorithm() error {
	if !isAsymmetricAlgorithm(c.signing.algorithm) {
		return nil
	}

	if c.signing.privateKeyPath == "" && c.signing.privateKeyPEM == "" {
		return ErrJWTMissingPrivateKey
	}

	if c.signing.publicKeyPath == "" && c.signing.publicKeyPEM == "" {
		return ErrJWTMissingPublicKey
	}

	if c.signing.privateKeyPEM != "" && c.signing.privateKeyPath != "" {
		return ErrJWTInvalidPrivateKeyConflict
	}

	if c.signing.publicKeyPEM != "" && c.signing.publicKeyPath != "" {
		return ErrJWTInvalidPublicKeyConflict
	}

	if c.signing.privateKeyPath != "" {
		privateKey, err := os.ReadFile(c.signing.privateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key file: %w", err)
		}
		c.signing.privateKeyPEM = string(privateKey)
	}

	if c.signing.publicKeyPath != "" {
		publicKey, err := os.ReadFile(c.signing.publicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read public key file: %w", err)
		}
		c.signing.publicKeyPEM = string(publicKey)
	}

	return nil
}

// Helper functions for algorithm validation
func isHMACAlgorithm(algorithm JWTAlgorithm) bool {
	return algorithm == JWTAlgorithmHS256 || algorithm == JWTAlgorithmHS384 || algorithm == JWTAlgorithmHS512
}

func isAsymmetricAlgorithm(algorithm JWTAlgorithm) bool {
	return isRSAAlgorithm(algorithm) || isECDSAAlgorithm(algorithm)
}

func isRSAAlgorithm(algorithm JWTAlgorithm) bool {
	return algorithm == JWTAlgorithmRS256 || algorithm == JWTAlgorithmRS384 || algorithm == JWTAlgorithmRS512
}

func isECDSAAlgorithm(algorithm JWTAlgorithm) bool {
	return algorithm == JWTAlgorithmES256 || algorithm == JWTAlgorithmES384 || algorithm == JWTAlgorithmES512
}

type jWTConfigOption func(*jWTConfig)

// WithJWTEnabled enables or disables JWT
func WithJWTEnabled(enabled bool) jWTConfigOption {
	return func(c *jWTConfig) {
		c.enabled = enabled
	}
}

// WithJWTAccessTokenDuration sets the duration of the access token
//
//	@default: 1hr
func WithJWTAccessTokenDuration(duration time.Duration) jWTConfigOption {
	return func(c *jWTConfig) {
		c.accessToken.duration = duration
	}
}

// WithJWTAccessTokenIssuer sets the issuer of the access token
func WithJWTAccessTokenIssuer(issuer string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.accessToken.issuer = issuer
	}
}

// WithJWTRefreshTokenEnabled enables or disables the refresh token
func WithJWTRefreshTokenEnabled(enabled bool) jWTConfigOption {
	return func(c *jWTConfig) {
		c.refreshToken.enabled = enabled
	}
}

// WithJWTRefreshTokenDuration sets the duration of the refresh token
func WithJWTRefreshTokenDuration(duration time.Duration) jWTConfigOption {
	return func(c *jWTConfig) {
		c.refreshToken.duration = duration
	}
}

// WithJWTRefreshTokenRotate sets whether to rotate the refresh token on use
func WithJWTRefreshTokenRotate(rotate bool) jWTConfigOption {
	return func(c *jWTConfig) {
		c.refreshToken.rotate = rotate
	}
}

// WithJWTSecret sets the secret of the JWT (HMAC algorithms only)
func WithJWTSecret(secret string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.signing.secret = secret
	}
}

// WithJWTAlgorithm sets the algorithm of the JWT (HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512)
func WithJWTAlgorithm(algorithm JWTAlgorithm) jWTConfigOption {
	return func(c *jWTConfig) {
		c.signing.algorithm = algorithm
	}
}

// WithJWTPrivateKeyPath sets the path to the private key (RSA/ECDSA algorithms only)
// To use the private key PEM, use WithJWTPrivateKeyPEM instead
func WithJWTPrivateKeyPath(privateKeyPath string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.signing.privateKeyPath = privateKeyPath
	}
}

// WithJWTPublicKeyPath sets the path to the public key (RSA/ECDSA algorithms only)
// To use the public key PEM, use WithJWTPublicKeyPEM instead
func WithJWTPublicKeyPath(publicKeyPath string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.signing.publicKeyPath = publicKeyPath
	}
}

// WithJWTPrivateKeyPEM sets the PEM encoded private key (RSA/ECDSA algorithms only)
// To use the private key file path, use WithJWTPrivateKeyPath instead
func WithJWTPrivateKeyPEM(privateKeyPEM string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.signing.privateKeyPEM = privateKeyPEM
	}
}

// WithJWTPublicKeyPEM sets the PEM encoded public key (RSA/ECDSA algorithms only)
// To use the public key file path, use WithJWTPublicKeyPath instead
func WithJWTPublicKeyPEM(publicKeyPEM string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.signing.publicKeyPEM = publicKeyPEM
	}
}

// WithClaimsSubjectField sets the field of the user that will be used as the subject of the JWT.
// This is usually the user's ID.
//
// To customize the subject field value, use WithClaimsSubjectValue instead
//
//	@default: "id"
func WithClaimsSubjectField(subjectField string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.claims.subjectField = subjectField
	}
}

// WithClaimsCustomClaims sets the fields of the user that will be included in the JWT
func WithClaimsCustomClaims(customClaims func(user *User) map[string]any) jWTConfigOption {
	return func(c *jWTConfig) {
		c.claims.customClaims = customClaims
	}
}

// WithClaimsSubjectValue sets the value of the subject field of the JWT.
//
// Useful for customizing the subject field value when the subject is not a direct field of the user
func WithClaimsSubjectValue(subjectValue func(user *User) string) jWTConfigOption {
	return func(c *jWTConfig) {
		c.claims.subjectValue = subjectValue
	}
}

// WithClaimsUserFromSubject sets the function to get the user from the subject field when the subject is not a direct field of the user
// i.e when WithClaimsSubjectValue is used
func WithClaimsUserFromSubject(userFromSubject func(subject string) (*User, error)) jWTConfigOption {
	return func(c *jWTConfig) {
		c.claims.userFromSubject = userFromSubject
	}
}
