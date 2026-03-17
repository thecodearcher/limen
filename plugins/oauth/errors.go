package oauth

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrOAuthStateInvalid                     = limen.NewLimenError("invalid or expired OAuth state", http.StatusBadRequest, nil)
	ErrMissingStateCookie                    = limen.NewLimenError("missing OAuth state cookie; ensure cookies are enabled", http.StatusBadRequest, nil)
	ErrAccountNotFound                       = limen.NewLimenError("account not found", http.StatusNotFound, nil)
	ErrProviderRequired                      = limen.NewLimenError("provider is required", http.StatusUnprocessableEntity, nil)
	ErrCannotUnlinkOnlyAccount               = limen.NewLimenError("cannot unlink the only account", http.StatusConflict, nil)
	ErrProviderNotFound                      = limen.NewLimenError("oauth provider not found", http.StatusNotFound, nil)
	ErrPKCEVerifierNotFound                  = limen.NewLimenError("PKCE verifier not found", http.StatusBadRequest, nil)
	ErrAccountAlreadyLinkedToAnotherUser     = limen.NewLimenError("this provider account is already linked to another user", http.StatusConflict, nil)
	ErrAccountCannotBeLinkedToDifferentEmail = limen.NewLimenError("user cannot be linked to this provider account because the email does not match", http.StatusConflict, nil)
	ErrNoRefreshToken                        = limen.NewLimenError("no refresh token available for this account", http.StatusBadRequest, nil)
)
