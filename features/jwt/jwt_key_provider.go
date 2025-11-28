package jwt

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// JWTKeyProvider defines the interface for managing signing keys
type jWTKeyProvider interface {
	getSigningKey() (any, error)         // Returns signing key ([]byte for HMAC, *rsa.PrivateKey, *ecdsa.PrivateKey)
	getVerificationKey() (any, error)    // Returns verification key (same as signing for HMAC, public key for RSA/ECDSA)
	getAlgorithm() JWTAlgorithm          // Returns algorithm name (HS256, RS256, ES256, etc.)
	getSigningMethod() jwt.SigningMethod // Returns the JWT signing method
}

// HMACKeyProvider implements JWTKeyProvider for HMAC algorithms
type hmacKeyProvider struct {
	secret    []byte
	algorithm JWTAlgorithm
}

// NewHMACKeyProvider creates a new HMAC key provider
func newHMACKeyProvider(config *signingConfig) (*hmacKeyProvider, error) {
	return &hmacKeyProvider{
		secret:    []byte(config.secret),
		algorithm: config.algorithm,
	}, nil
}

func (h *hmacKeyProvider) getSigningKey() (any, error) {
	return h.secret, nil
}

func (h *hmacKeyProvider) getVerificationKey() (any, error) {
	return h.getSigningKey()
}

func (h *hmacKeyProvider) getAlgorithm() JWTAlgorithm {
	return h.algorithm
}

func (h *hmacKeyProvider) getSigningMethod() jwt.SigningMethod {
	switch h.algorithm {
	case "HS256":
		return jwt.SigningMethodHS256
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return nil
	}
}

// RSAKeyProvider implements JWTKeyProvider for RSA algorithms
type asymmetricKeyProvider[T, U any] struct {
	privateKey T
	publicKey  U
	algorithm  JWTAlgorithm
}

// NewRSAKeyProvider creates a new RSA key provider
func newAsymmetricKeyProvider[T, U any](config *signingConfig) (*asymmetricKeyProvider[T, U], error) {
	var privateKey T
	var publicKey U

	if isRSAAlgorithm(config.algorithm) {
		rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(config.privateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
		}
		privateKey = any(rsaPrivateKey).(T)

		rsaPublicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(config.publicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
		}
		publicKey = any(rsaPublicKey).(U)
	}

	if isECDSAAlgorithm(config.algorithm) {
		ecdsaPrivateKey, err := jwt.ParseECPrivateKeyFromPEM([]byte(config.privateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse ECDSA private key: %w", err)
		}
		privateKey = any(ecdsaPrivateKey).(T)

		ecdsaPublicKey, err := jwt.ParseECPublicKeyFromPEM([]byte(config.publicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse ECDSA public key: %w", err)
		}
		publicKey = any(ecdsaPublicKey).(U)
	}

	return &asymmetricKeyProvider[T, U]{
		privateKey: privateKey,
		publicKey:  publicKey,
		algorithm:  config.algorithm,
	}, nil
}

func (r *asymmetricKeyProvider[T, U]) getSigningKey() (any, error) {
	return r.privateKey, nil
}

func (r *asymmetricKeyProvider[T, U]) getVerificationKey() (any, error) {
	return r.publicKey, nil
}

func (r *asymmetricKeyProvider[T, U]) getAlgorithm() JWTAlgorithm {
	return r.algorithm
}

func (r *asymmetricKeyProvider[T, U]) getSigningMethod() jwt.SigningMethod {
	switch r.algorithm {
	case "RS256":
		return jwt.SigningMethodRS256
	case "RS384":
		return jwt.SigningMethodRS384
	case "RS512":
		return jwt.SigningMethodRS512
	case "ES256":
		return jwt.SigningMethodES256
	case "ES384":
		return jwt.SigningMethodES384
	case "ES512":
		return jwt.SigningMethodES512
	default:
		return nil
	}
}

// newKeyProvider creates the appropriate key provider based on configuration
func newKeyProvider(config *signingConfig) (jWTKeyProvider, error) {
	switch {
	case isHMACAlgorithm(config.algorithm):
		return newHMACKeyProvider(config)
	case isRSAAlgorithm(config.algorithm):
		return newAsymmetricKeyProvider[*rsa.PrivateKey, *rsa.PublicKey](config)
	case isECDSAAlgorithm(config.algorithm):
		return newAsymmetricKeyProvider[*ecdsa.PrivateKey, *ecdsa.PublicKey](config)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", config.algorithm)
	}
}
