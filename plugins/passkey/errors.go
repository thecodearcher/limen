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
