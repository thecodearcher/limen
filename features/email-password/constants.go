package emailpassword

import "errors"

const (
	defaultMinPasswordLength        = 4
	defaultPasswordRequireUppercase = true
	defaultPasswordRequireNumbers   = true
	defaultPasswordRequireSymbols   = false
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailNotFound   = errors.New("email not found")
)
