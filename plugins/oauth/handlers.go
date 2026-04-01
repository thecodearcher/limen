package oauth

import (
	"net/http"
	"net/url"

	"github.com/thecodearcher/limen"
)

type oauthHandlers struct {
	plugin    *oauthPlugin
	responder *limen.Responder
}

func newOAuthHandlers(plugin *oauthPlugin, httpCore *limen.LimenHTTPCore) *oauthHandlers {
	return &oauthHandlers{
		plugin:    plugin,
		responder: httpCore.Responder,
	}
}

func (h *oauthHandlers) SignInWithOAuth(w http.ResponseWriter, r *http.Request) {
	providerName := limen.GetParam(r, "provider")

	request := &OAuthAuthorizeURLData{
		RedirectURI:      r.URL.Query().Get("redirect_uri"),
		ErrorRedirectURI: r.URL.Query().Get("error_redirect_uri"),
	}
	url, cookieValue, err := h.plugin.GetAuthorizationURL(r.Context(), providerName, request)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.setStateCookie(w, cookieValue)
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"url": url})
}

func (h *oauthHandlers) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := limen.GetParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	callbackErr := callbackErrorFromQuery(r.URL.Query())

	cookieValue, err := h.plugin.cookies.Get(r, h.plugin.config.cookieName)
	if err != nil {
		h.handleCallbackResponse(w, r, nil, nil, nil, ErrMissingStateCookie)
		return
	}

	h.clearStateCookie(w)

	result, stateData, err := h.plugin.AuthenticateWithProvider(r.Context(), providerName, code, state, cookieValue, callbackErr)
	if err != nil {
		h.handleCallbackResponse(w, r, stateData, nil, nil, err)
		return
	}

	var sessionResult *limen.SessionResult
	if stateData[linkUserIdKey] == nil {
		sessionResult, err = h.plugin.core.CreateSession(r.Context(), r, w, result)
		if err != nil {
			h.handleCallbackResponse(w, r, stateData, nil, nil, err)
			return
		}
	}

	h.handleCallbackResponse(w, r, stateData, result, sessionResult, err)
}

// LinkAccountWithOAuth initiates the OAuth flow for linking a provider to the current user's account.
func (h *oauthHandlers) LinkAccountWithOAuth(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")
	data := map[string]any{
		linkUserIdKey: session.User.ID,
	}
	request := &OAuthAuthorizeURLData{
		AdditionalData:   data,
		RedirectURI:      r.URL.Query().Get("redirect_uri"),
		ErrorRedirectURI: r.URL.Query().Get("error_redirect_uri"),
	}

	url, cookieValue, err := h.plugin.GetAuthorizationURL(r.Context(), providerName, request)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.setStateCookie(w, cookieValue)
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"url": url})
}

func (h *oauthHandlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	accounts, err := h.plugin.ListAccountsForUser(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, limen.SerializeAll(h.plugin.accountSchema, accounts))
}

func (h *oauthHandlers) UnlinkAccount(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")

	err = h.plugin.UnlinkAccount(r.Context(), session.User, providerName)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *oauthHandlers) GetTokens(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")

	tokens, err := h.plugin.GetAccessToken(r.Context(), session.User.ID, providerName)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, tokens)
}

func (h *oauthHandlers) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")

	tokens, err := h.plugin.RefreshAccessToken(r.Context(), session.User.ID, providerName)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, tokens)
}

func (h *oauthHandlers) handleCallbackResponse(w http.ResponseWriter, r *http.Request, stateData map[string]any, authResult *limen.AuthenticationResult, sessionResult *limen.SessionResult, err error) {
	if (h.plugin.config.disableRedirect && err != nil) || stateData == nil {
		h.responder.Error(w, r, err)
		return
	}

	if h.plugin.config.disableRedirect {
		h.responder.SessionResponse(w, r, h.plugin.core, authResult, sessionResult)
		return
	}

	redirectURI, _ := stateData[redirectURIKey].(string)
	errorRedirectURI, _ := stateData[errorRedirectURIKey].(string)
	if err != nil && errorRedirectURI != "" {
		redirectURI = errorRedirectURI
	}

	if err != nil {
		redirectURI = h.buildErrorRedirectURL(redirectURI, err)
	}

	h.responder.RedirectWithSession(w, r, redirectURI, sessionResult)
}

// buildErrorRedirectURL appends error query parameters to the redirect URL.
// When the error carries structured OAuth details (code, error_description),
// those are forwarded as separate params per RFC 6749. Otherwise the error
// message is placed in a single "error" param.
func (h *oauthHandlers) buildErrorRedirectURL(redirectURI string, err error) string {
	ae := limen.ToLimenError(err)
	if details, ok := ae.Details().(map[string]string); ok {
		code := details["code"]
		if code != "" {
			return appendOAuthErrorParams(redirectURI, code, details["error_description"])
		}
	}

	return appendOAuthErrorParams(redirectURI, ae.Error(), "")
}

// FormPostCallback handles OAuth callbacks delivered via response_mode=form_post.
// The IdP POSTs code/state/error as application/x-www-form-urlencoded. Rather than
// processing the POST directly (which lacks cookies), we extract
// the form values and 303 redirect to the same path as a GET with query parameters.
// The browser follows the redirect as a same-site navigation, attaching cookies normally.
func (h *oauthHandlers) FormPostCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	// Preserve all existing query params and merge all form-post params.
	target := url.URL{
		Path:     r.URL.Path,
		RawQuery: r.Form.Encode(),
	}
	http.Redirect(w, r, target.String(), http.StatusSeeOther)
}

func (h *oauthHandlers) setStateCookie(w http.ResponseWriter, value string) {
	h.plugin.cookies.Set(w, h.plugin.config.cookieName, value, int(h.plugin.config.cookieTTL.Seconds()))
}

func (h *oauthHandlers) clearStateCookie(w http.ResponseWriter) {
	h.plugin.cookies.Delete(w, h.plugin.config.cookieName)
}
