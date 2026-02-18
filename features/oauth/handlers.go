package oauth

import (
	"fmt"
	"net/http"

	"github.com/thecodearcher/aegis"
)

type oauthHandlers struct {
	feature   *oauthFeature
	responder *aegis.Responder
}

func newOAuthHandlers(feature *oauthFeature, httpCore *aegis.AegisHTTPCore) *oauthHandlers {
	return &oauthHandlers{
		feature:   feature,
		responder: httpCore.Responder,
	}
}

func (h *oauthHandlers) SignInWithOAuth(w http.ResponseWriter, r *http.Request) {
	providerName := aegis.GetParam(r, "provider")

	request := &OAuthAuthorizeURLData{
		AdditionalData:   h.queryToMap(r),
		RedirectURI:      r.URL.Query().Get("redirect_uri"),
		ErrorRedirectURI: r.URL.Query().Get("error_redirect_uri"),
	}

	url, cookieValue, err := h.feature.GetAuthorizationURL(r.Context(), providerName, request)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.setStateCookie(w, cookieValue)
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"url": url})
}

func (h *oauthHandlers) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := aegis.GetParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	cookie, err := r.Cookie(h.feature.config.cookieName)
	if err != nil {
		h.handleLinkAccountResponse(w, r, nil, nil, nil, ErrMissingStateCookie)
		return
	}

	h.clearStateCookie(w)

	result, stateData, err := h.feature.AuthenticateWithProvider(r.Context(), providerName, code, state, cookie.Value)
	if err != nil {
		h.handleLinkAccountResponse(w, r, stateData, nil, nil, err)
		return
	}

	var sessionResult *aegis.SessionResult
	if stateData[linkUserIdKey] == nil {
		sessionResult, err = h.feature.core.SessionManager.CreateSession(r.Context(), r, result)
		if err != nil {
			h.handleLinkAccountResponse(w, r, stateData, nil, nil, err)
			return
		}
	}

	h.handleLinkAccountResponse(w, r, stateData, result, sessionResult, err)
}

// LinkAccountWithOAuth initiates the OAuth flow for linking a provider to the current user's account.
func (h *oauthHandlers) LinkAccountWithOAuth(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := aegis.GetParam(r, "provider")
	data := h.queryToMap(r)
	data[linkUserIdKey] = session.User.ID
	request := &OAuthAuthorizeURLData{
		AdditionalData:   data,
		RedirectURI:      r.URL.Query().Get("redirect_uri"),
		ErrorRedirectURI: r.URL.Query().Get("error_redirect_uri"),
	}

	url, cookieValue, err := h.feature.GetAuthorizationURL(r.Context(), providerName, request)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.setStateCookie(w, cookieValue)
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"url": url})
}

func (h *oauthHandlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	accounts, err := h.feature.ListAccountsForUser(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, aegis.SerializeAll(h.feature.accountSchema, accounts))
}

func (h *oauthHandlers) UnlinkAccount(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := aegis.GetParam(r, "provider")

	err = h.feature.UnlinkAccount(r.Context(), session.User, providerName)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *oauthHandlers) handleLinkAccountResponse(w http.ResponseWriter, r *http.Request, stateData map[string]any, authResult *aegis.AuthenticationResult, sessionResult *aegis.SessionResult, err error) {
	if h.feature.config.disableRedirect && (err != nil || stateData == nil) {
		h.responder.Error(w, r, err)
		return
	}

	if h.feature.config.disableRedirect {
		h.responder.SessionResponse(w, r, h.feature.core, authResult, sessionResult)
		return
	}

	redirectURI, _ := stateData[redirectURIKey].(string)
	errorRedirectURI, _ := stateData[errorRedirectURIKey].(string)
	if err != nil && errorRedirectURI != "" {
		redirectURI = errorRedirectURI
	}

	if err != nil {
		redirectURI = fmt.Sprintf("%s?error=%s", redirectURI, err.Error())
	}

	h.responder.RedirectWithSession(w, r, redirectURI, sessionResult)
}

func (h *oauthHandlers) queryToMap(r *http.Request) map[string]any {
	data := make(map[string]any)
	for key, value := range r.URL.Query() {
		if key == "redirect_uri" || key == "error_redirect_uri" {
			continue
		}
		data[key] = value[0]
	}
	return data
}

func (h *oauthHandlers) setStateCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.feature.config.cookieName,
		Value:    value,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.feature.config.cookieTTL.Seconds()),
		Path:     "/",
	})
}

func (h *oauthHandlers) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.feature.config.cookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Path:     "/",
	})
}
