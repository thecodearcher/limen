package sessionjwt

import (
	"net/http"

	"github.com/thecodearcher/aegis"
)

var (
	ErrInvalidAccessToken    = aegis.NewAegisError("invalid or expired access token", http.StatusUnauthorized, nil)
	ErrInvalidRefreshToken   = aegis.NewAegisError("invalid or expired refresh token", http.StatusUnauthorized, nil)
	ErrRefreshTokenReuse     = aegis.NewAegisError("refresh token reuse detected, family revoked", http.StatusUnauthorized, nil)
	ErrTokenRevoked          = aegis.NewAegisError("token has been revoked", http.StatusUnauthorized, nil)
	ErrMissingRefreshToken   = aegis.NewAegisError("refresh token is required", http.StatusBadRequest, nil)
	ErrRefreshTokensDisabled = aegis.NewAegisError("refresh tokens are not enabled", http.StatusBadRequest, nil)
)
