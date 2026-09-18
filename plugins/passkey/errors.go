package passkey

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/thecodearcher/limen"
)

var (
	ErrChallengeMissing   = limen.NewLimenError("passkey challenge is missing or expired", http.StatusBadRequest, nil)
	ErrChallengeInvalid   = limen.NewLimenError("passkey challenge is missing or expired", http.StatusBadRequest, nil)
	ErrVerificationFailed = limen.NewLimenError("passkey verification failed", http.StatusBadRequest, nil)
	ErrUnknownPasskey     = limen.NewLimenError("passkey not recognized", http.StatusUnauthorized, nil)
	ErrPasskeyNotFound    = limen.NewLimenError("passkey not found", http.StatusNotFound, nil)

	ErrHandleModeRequired = limen.NewLimenError(
		"passkey needs opaque user IDs: enable public IDs or WithAllowInternalUserIDAsHandle",
		http.StatusInternalServerError,
		nil,
	)
	ErrIDGeneratorRequired       = limen.NewLimenError("passkey registration needs RegistrationIntent.ID or an ID generator", http.StatusInternalServerError, nil)
	ErrRegistrationHooksRequired = limen.NewLimenError(
		"passkey public registration requires WithPrepareRegistration and WithCreateRegistrationUser",
		http.StatusInternalServerError,
		nil,
	)
	ErrRegistrationContextRequired = limen.NewLimenError("passkey registration context is required", http.StatusBadRequest, nil)
	ErrRegistrationIntentInvalid   = limen.NewLimenError("passkey registration intent is invalid", http.StatusBadRequest, nil)
	ErrRegistrationUserMissing     = limen.NewLimenError("passkey registration did not create a user", http.StatusInternalServerError, nil)
	ErrHandleMismatch              = limen.NewLimenError("passkey user handle is missing or does not match the reserved handle", http.StatusInternalServerError, nil)
	ErrSessionRequired             = limen.NewLimenError("passkey registration requires a session", http.StatusUnauthorized, nil)
)

func toPasskeyError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, limen.ErrRecordNotFound) {
		return ErrUnknownPasskey
	}

	var limenErr *limen.LimenError
	if errors.As(err, &limenErr) {
		return limenErr
	}

	var unknown *protocol.ErrorUnknownCredential
	if errors.As(err, &unknown) {
		return ErrUnknownPasskey
	}

	var protoErr *protocol.Error
	if errors.As(err, &protoErr) {
		return mapProtocolError(protoErr)
	}

	return err
}

func mapProtocolError(err *protocol.Error) error {
	switch {
	case strings.Contains(err.Details, "Failed to lookup Client-side Discoverable Credential"),
		strings.Contains(err.Details, "Unable to find the credential"):
		return ErrUnknownPasskey
	case err.Type == "challenge_mismatch",
		strings.Contains(err.Details, "Session has Expired"),
		strings.Contains(err.Details, "Session was not initiated as a client-side discoverable login"):
		return ErrChallengeInvalid
	default:
		return ErrVerificationFailed
	}
}
