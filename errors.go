package limen

import (
	"errors"
	"net/http"
)

type LimenError struct {
	message string
	details any
	status  int
}

var (
	ErrDatabaseAdapterRequired = errors.New("database adapter is required")
	ErrPluginNotFound          = errors.New("plugin not found")
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
	ErrInvalidConfiguration    = errors.New("invalid configuration")
	ErrRecordNotFound          = errors.New("record not found")
	ErrEmptyText               = errors.New("text is empty and cannot be encrypted or decrypted")
)

// Session-specific errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionInvalid  = errors.New("session is invalid")
)

// Rate limiting errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrRateLimitNotFound = errors.New("rate limit not found")
)

// Verification errors
var (
	ErrVerificationTokenInvalid = errors.New("verification token is invalid")
)

func NewLimenError(message string, status int, details any) *LimenError {
	return &LimenError{message: message, details: details, status: status}
}

func (e *LimenError) Error() string {
	return e.message
}

func (e *LimenError) Details() any {
	return e.details
}

func (e *LimenError) Status() int {
	return e.status
}

func ToLimenError(err error) *LimenError {
	var limenErr *LimenError
	if errors.As(err, &limenErr) {
		return err.(*LimenError)
	}

	if errors.Is(err, ErrRecordNotFound) {
		return NewLimenError(err.Error(), http.StatusNotFound, err)
	}

	return NewLimenError(err.Error(), http.StatusInternalServerError, err)
}
