package sessionjwt

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/aegis"
)

type jwtHandlers struct {
	plugin   *sessionJWTPlugin
	httpCore *aegis.AegisHTTPCore
}

func newJWTHandlers(plugin *sessionJWTPlugin, httpCore *aegis.AegisHTTPCore) *jwtHandlers {
	return &jwtHandlers{
		plugin:   plugin,
		httpCore: httpCore,
	}
}

// Refresh exchanges a valid refresh token for a new JWT access token.
// When rotation is enabled the old refresh token is consumed and a new one
// is returned in the response body.
func (h *jwtHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	sessionResult, user, err := h.plugin.RefreshAccessToken(r)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenReuse) {
			h.httpCore.Responder.Error(w, r, ErrInvalidRefreshToken)
			return
		}
		h.httpCore.Responder.Error(w, r, err)
		return
	}

	authResult := &aegis.AuthenticationResult{User: user}
	h.httpCore.Responder.SessionResponse(w, r, h.plugin.core, authResult, sessionResult)
}
