package emailpassword

import "errors"

const (
	defaultMinPasswordLength        = 4
	defaultPasswordRequireUppercase = true
	defaultPasswordRequireNumbers   = true
	defaultPasswordRequireSymbols   = false
)

var DefaultPasswordHasherConfig = PasswordHasherConfig{
	Time:      3,
	MemoryKiB: 64 * 1024,
	Parallel:  1,
	SaltLen:   16,
	KeyLen:    32,
}

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailNotFound   = errors.New("email not found")
)
