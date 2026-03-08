package sessionjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// keyProvider abstracts signing-key resolution for different JWT algorithm
// families. Each implementation handles PEM parsing, type validation, and
// public-key derivation for its algorithm.
type keyProvider interface {
	Name() string
	ResolveSigningKey(key any) (any, error)
	ResolveVerificationKey(key any) (any, error)
	DerivePublicKey(privateKey any) any
}

// resolveKeys validates and normalises signing/verification keys for the
// configured signing method. PEM-encoded strings are automatically parsed
// into their native crypto types. For asymmetric methods the public key is
// derived from the private key when no explicit verification key is provided.
func (c *config) resolveKeys(coreSecret []byte) error {
	switch c.signingMethod.(type) {
	case *jwt.SigningMethodHMAC:
		return c.resolveHMACKeys(coreSecret)
	case *jwt.SigningMethodRSA, *jwt.SigningMethodRSAPSS:
		return c.resolveAsymmetricKeys(rsaKeyProvider{})
	case *jwt.SigningMethodECDSA:
		return c.resolveAsymmetricKeys(ecdsaKeyProvider{})
	case *jwt.SigningMethodEd25519:
		return c.resolveAsymmetricKeys(eddsaKeyProvider{})
	default:
		return fmt.Errorf("session-jwt: unsupported signing method %q", c.signingMethod.Alg())
	}
}

func (c *config) resolveAsymmetricKeys(kp keyProvider) error {
	alg := c.signingMethod.Alg()

	if c.signingKey == nil {
		return fmt.Errorf("session-jwt: %s requires a %s key pair; use WithSigningKey", alg, kp.Name())
	}

	privKey, err := kp.ResolveSigningKey(c.signingKey)
	if err != nil {
		return fmt.Errorf("session-jwt: %s signing key: %w", alg, err)
	}
	c.signingKey = privKey

	if c.verificationKey == nil {
		c.verificationKey = kp.DerivePublicKey(privKey)
		return nil
	}

	pubKey, err := kp.ResolveVerificationKey(c.verificationKey)
	if err != nil {
		return fmt.Errorf("session-jwt: %s verification key: %w", alg, err)
	}
	c.verificationKey = pubKey
	return nil
}

func (c *config) resolveHMACKeys(coreSecret []byte) error {
	alg := c.signingMethod.Alg()

	if c.signingKey == nil {
		if len(coreSecret) == 0 {
			return fmt.Errorf("session-jwt: %s requires a signing key; use WithSigningKey or set Config.SigningSecret", alg)
		}
		c.signingKey = coreSecret
		c.verificationKey = coreSecret
		return nil
	}

	key, ok := c.signingKey.(string)
	if !ok {
		return fmt.Errorf("session-jwt: %s signing key must be a string, got %T", alg, c.signingKey)
	}
	if key == "" {
		return fmt.Errorf("session-jwt: %s signing key must not be empty", alg)
	}

	c.signingKey = []byte(key)
	c.verificationKey = c.signingKey
	return nil
}

// ---------------------------------------------------------------------------
// RSA (RS256, RS384, RS512, PS256, PS384, PS512)
// ---------------------------------------------------------------------------

type rsaKeyProvider struct{}

func (rsaKeyProvider) Name() string {
	return "RSA"
}

func (rsaKeyProvider) ResolveSigningKey(key any) (any, error) {
	switch k := key.(type) {
	case string:
		return jwt.ParseRSAPrivateKeyFromPEM([]byte(k))
	case *rsa.PrivateKey:
		return k, nil
	default:
		return nil, fmt.Errorf("must be a PEM string or *rsa.PrivateKey; got %T", key)
	}
}

func (rsaKeyProvider) ResolveVerificationKey(key any) (any, error) {
	switch k := key.(type) {
	case string:
		return jwt.ParseRSAPublicKeyFromPEM([]byte(k))
	case *rsa.PublicKey:
		return k, nil
	default:
		return nil, fmt.Errorf("must be a PEM string or *rsa.PublicKey; got %T", key)
	}
}

func (rsaKeyProvider) DerivePublicKey(privateKey any) any {
	return &privateKey.(*rsa.PrivateKey).PublicKey
}

// ---------------------------------------------------------------------------
// ECDSA (ES256, ES384, ES512)
// ---------------------------------------------------------------------------

type ecdsaKeyProvider struct{}

func (ecdsaKeyProvider) Name() string {
	return "ECDSA"
}

func (ecdsaKeyProvider) ResolveSigningKey(key any) (any, error) {
	switch k := key.(type) {
	case string:
		return jwt.ParseECPrivateKeyFromPEM([]byte(k))
	case *ecdsa.PrivateKey:
		return k, nil
	default:
		return nil, fmt.Errorf("must be a PEM string or *ecdsa.PrivateKey; got %T", key)
	}
}

func (ecdsaKeyProvider) ResolveVerificationKey(key any) (any, error) {
	switch k := key.(type) {
	case string:
		return jwt.ParseECPublicKeyFromPEM([]byte(k))
	case *ecdsa.PublicKey:
		return k, nil
	default:
		return nil, fmt.Errorf("must be a PEM string or *ecdsa.PublicKey; got %T", key)
	}
}

func (ecdsaKeyProvider) DerivePublicKey(privateKey any) any {
	return &privateKey.(*ecdsa.PrivateKey).PublicKey
}

// ---------------------------------------------------------------------------
// EdDSA (Ed25519)
// ---------------------------------------------------------------------------

type eddsaKeyProvider struct{}

func (eddsaKeyProvider) Name() string {
	return "Ed25519"
}

func (eddsaKeyProvider) ResolveSigningKey(key any) (any, error) {
	switch k := key.(type) {
	case string:
		return jwt.ParseEdPrivateKeyFromPEM([]byte(k))
	case ed25519.PrivateKey:
		return k, nil
	default:
		return nil, fmt.Errorf("must be a PEM string or ed25519.PrivateKey; got %T", key)
	}
}

func (eddsaKeyProvider) ResolveVerificationKey(key any) (any, error) {
	switch k := key.(type) {
	case string:
		return jwt.ParseEdPublicKeyFromPEM([]byte(k))
	case ed25519.PublicKey:
		return k, nil
	default:
		return nil, fmt.Errorf("must be a PEM string or ed25519.PublicKey; got %T", key)
	}
}

func (eddsaKeyProvider) DerivePublicKey(privateKey any) any {
	return privateKey.(ed25519.PrivateKey).Public()
}
