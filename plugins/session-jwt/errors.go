package sessionjwt

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrInvalidAccessToken    = limen.NewLimenError("invalid or expired access token", http.StatusUnauthorized, nil)
	ErrInvalidRefreshToken   = limen.NewLimenError("invalid or expired refresh token", http.StatusUnauthorized, nil)
	ErrRefreshTokenReuse     = limen.NewLimenError("refresh token reuse detected, family revoked", http.StatusUnauthorized, nil)
	ErrTokenRevoked          = limen.NewLimenError("token has been revoked", http.StatusUnauthorized, nil)
	ErrStaleAccessToken      = limen.NewLimenError("access token has been superseded, refresh and retry", http.StatusConflict, nil)
	ErrMissingRefreshToken   = limen.NewLimenError("refresh token is required", http.StatusBadRequest, nil)
	ErrRefreshTokensDisabled = limen.NewLimenError("refresh tokens are not enabled", http.StatusBadRequest, nil)
	ErrMissingUserID         = limen.NewLimenError("user id is required when reconstructing the user from claims", http.StatusInternalServerError, nil)
)
