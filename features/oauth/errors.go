package oauth

import (
	"net/http"

	"github.com/thecodearcher/aegis"
)

var (
	ErrOAuthStateInvalid                     = aegis.NewAegisError("invalid or expired OAuth state", http.StatusBadRequest, nil)
	ErrOAuthStateDataTooLarge                = aegis.NewAegisError("OAuth state data exceeds maximum size; use less data or the database state store", http.StatusBadRequest, nil)
	ErrMissingStateCookie                    = aegis.NewAegisError("missing OAuth state cookie; ensure cookies are enabled", http.StatusBadRequest, nil)
	ErrAccountNotFound                       = aegis.NewAegisError("account not found", http.StatusNotFound, nil)
	ErrProviderRequired                      = aegis.NewAegisError("provider is required", http.StatusUnprocessableEntity, nil)
	ErrCannotUnlinkOnlyAccount               = aegis.NewAegisError("cannot unlink the only account", http.StatusConflict, nil)
	ErrProviderNotFound                      = aegis.NewAegisError("oauth provider not found", http.StatusNotFound, nil)
	ErrPKCEVerifierNotFound                  = aegis.NewAegisError("PKCE verifier not found", http.StatusBadRequest, nil)
	ErrAccountAlreadyLinkedToAnotherUser     = aegis.NewAegisError("this provider account is already linked to another user", http.StatusConflict, nil)
	ErrAccountCannotBeLinkedToDifferentEmail = aegis.NewAegisError("user cannot be linked to this provider account because the email does not match", http.StatusConflict, nil)
)
