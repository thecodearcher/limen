package credentialpassword

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// passwordHasherConfig defines the configuration parameters for Argon2id hashing.
type passwordHasherConfig struct {
	time      uint32 // Number of iterations (t parameter)
	memoryKiB uint32 // Memory usage in KiB (m parameter)
	Parallel  uint8  // Number of parallel threads (p parameter)
	saltLen   uint32 // Salt length in bytes
	keyLen    uint32 // Output key length in bytes
}

// passwordHasher handles the hashing and verification of passwords.
type passwordHasher struct {
	passwordHasherConfig
}

// hashInfo contains the parsed components of a PHC-formatted hash.
type hashInfo struct {
	algorithm string
	version   int
	memory    uint32
	time      uint32
	parallel  uint8
	salt      []byte
	hash      []byte
	keyLength uint32
}

// newPasswordHasher provides secure baseline parameters for Argon2id.
//
// Security notes:
// - ASVS/RFC9106-second recommendation: t=3 ⇒ m ≥ 64MiB, p=1
func newPasswordHasher(c passwordHasherConfig) *passwordHasher {
	return &passwordHasher{
		passwordHasherConfig: c,
	}
}

// hashPassword creates a Argon2id hash from a password.
func (p *passwordHasher) hashPassword(password []byte) (string, error) {
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := p.hash(password, salt)

	encoder := base64.RawStdEncoding
	saltB64 := encoder.EncodeToString(salt)
	hashB64 := encoder.EncodeToString(hash)

	phcString := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.memoryKiB,
		p.time,
		p.Parallel,
		saltB64,
		hashB64,
	)

	return phcString, nil
}

// verifyPassword compares a password against a Argon2id hash.
func (p *passwordHasher) verifyPassword(password []byte, hash string) (bool, error) {
	hashInfo, err := p.parseHash(hash)
	if err != nil {
		return false, fmt.Errorf("failed to parse hash: %w", err)
	}

	if hashInfo.algorithm != "argon2id" || hashInfo.version != 19 {
		return false, errors.New("unsupported Argon2 variant or version")
	}

	computedHash := p.hash(password, hashInfo.salt)

	// Compare computed hash with stored hash using constant-time comparison
	// to prevent timing attacks
	matches := subtle.ConstantTimeCompare(computedHash, hashInfo.hash) == 1
	return matches, nil
}

// Generate Argon2id hash using the PasswordHasher parameters
func (p *passwordHasher) hash(password []byte, salt []byte) []byte {
	return argon2.IDKey(
		password,
		salt,
		p.time,
		p.memoryKiB,
		p.Parallel,
		p.keyLen,
	)
}

// parseHash parses a hash string and extracts its components.
//
// PHC format: $<algorithm>$v=<version>$<params>$<salt>$<hash>
func (p *passwordHasher) parseHash(hashString string) (*hashInfo, error) {
	// Split by $ to get the components
	// Expected format: ["", "argon2id", "v=19", "m=...,t=...,p=...", "<salt>", "<hash>"]
	parts := strings.Split(hashString, "$")
	if len(parts) != 6 {
		return nil, errors.New("invalid PHC format: expected 6 parts")
	}

	hashInfo := &hashInfo{
		algorithm: parts[1],
	}

	if !strings.HasPrefix(parts[2], "v=") {
		return nil, errors.New("missing version in PHC string")
	}

	versionStr := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %w", err)
	}
	hashInfo.version = int(version)

	if err := p.parseParams(parts[3], hashInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PHC parameters: %w", err)
	}

	decoder := base64.RawStdEncoding

	salt, err := decoder.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	hashInfo.salt = salt

	hash, err := decoder.DecodeString(parts[5])
	if err != nil {
		return nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	hashInfo.hash = hash
	hashInfo.keyLength = uint32(len(hash))

	return hashInfo, nil
}

// parseParams parses the parameters section of a hash string.
func (p *passwordHasher) parseParams(paramsStr string, hashInfo *hashInfo) error {
	paramPairs := strings.Split(paramsStr, ",")

	for _, pair := range paramPairs {
		keyValue := strings.SplitN(pair, "=", 2)
		if len(keyValue) != 2 {
			return errors.New("malformed parameter format")
		}

		key := keyValue[0]
		valueStr := keyValue[1]

		value, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid parameter value for %s: %w", key, err)
		}

		switch key {
		case "m":
			hashInfo.memory = uint32(value)
		case "t":
			hashInfo.time = uint32(value)
		case "p":
			hashInfo.parallel = uint8(value)
		default:
			return fmt.Errorf("unknown parameter: %s", key)
		}
	}

	return nil
}

type PasswordHasherConfigOption func(*passwordHasherConfig)

// DefaultPasswordHasherConfig creates a new password hasher configuration with default values
// the default configuration follows the RFC9106-second recommendation:
// t=3 iterations, m ≥ 64MiB memory, p=4 lanes, s=128-bits salt and k=256-bits tag size.
//
// @see https://datatracker.ietf.org/doc/html/rfc9106#section-7.4
//
// These parameters provide a good balance between security and performance.
func DefaultPasswordHasherConfig(opts ...PasswordHasherConfigOption) passwordHasherConfig {
	config := passwordHasherConfig{
		time:      3,
		memoryKiB: 64 * 1024,
		Parallel:  4,
		saltLen:   16,
		keyLen:    32,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// WithPasswordHasherTime sets the number of iterations (t parameter)
func WithPasswordHasherTime(time uint32) PasswordHasherConfigOption {
	return func(c *passwordHasherConfig) {
		c.time = time
	}
}

// WithPasswordHasherMemoryKiB sets the memory usage in KiB (m parameter)
func WithPasswordHasherMemoryKiB(memoryKiB uint32) PasswordHasherConfigOption {
	return func(c *passwordHasherConfig) {
		c.memoryKiB = memoryKiB
	}
}

// WithPasswordHasherParallel sets the number of parallel threads (p parameter)
func WithPasswordHasherParallel(parallel uint8) PasswordHasherConfigOption {
	return func(c *passwordHasherConfig) {
		c.Parallel = parallel
	}
}

// WithPasswordHasherSaltLen sets the salt length in bytes
func WithPasswordHasherSaltLen(saltLen uint32) PasswordHasherConfigOption {
	return func(c *passwordHasherConfig) {
		c.saltLen = saltLen
	}
}

// WithPasswordHasherKeyLen sets the output key length in bytes
func WithPasswordHasherKeyLen(keyLen uint32) PasswordHasherConfigOption {
	return func(c *passwordHasherConfig) {
		c.keyLen = keyLen
	}
}
