package magiclink

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/thecodearcher/limen"
)

type magicLinkHandlers struct {
	plugin    *magicLinkPlugin
	httpCore  *limen.LimenHTTPCore
	responder *limen.Responder
}

func newMagicLinkHandlers(plugin *magicLinkPlugin, httpCore *limen.LimenHTTPCore) *magicLinkHandlers {
	return &magicLinkHandlers{
		plugin:    plugin,
		httpCore:  httpCore,
		responder: httpCore.Responder,
	}
}

func (h *magicLinkHandlers) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("email", data["email"]).Email("email", data["email"])
	})
	if body == nil {
		return
	}

	redirectURI, newUserRedirectURI, errorRedirectURI, err := h.validateRedirectURLs(body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	additionalData, ok := body["meta"].(map[string]any)
	if !ok {
		additionalData = map[string]any{}
	}

	_, err = h.plugin.RequestMagicLink(r.Context(), body["email"].(string), &RequestMagicLinkOptions{
		RedirectURI:        redirectURI,
		NewUserRedirectURI: newUserRedirectURI,
		ErrorRedirectURI:   errorRedirectURI,
		AdditionalData:     additionalData,
	})
	if err != nil && !errors.Is(err, ErrEmailNotFound) {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, "A magic link has been sent to your email address")
}

func (h *magicLinkHandlers) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	token, err := h.resolveVerifyMagicLinkParams(r.URL.Query())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	result, state, err := h.plugin.VerifyMagicLink(r.Context(), token)
	callbackURL, errorCallbackURL := h.resolveCallbackURLs(state)
	if err != nil {
		h.handleVerifyMagicLinkError(w, r, callbackURL, errorCallbackURL, err)
		return
	}

	sessionResult, err := h.plugin.core.CreateSession(r.Context(), r, w, result)
	if err != nil {
		h.handleVerifyMagicLinkError(w, r, callbackURL, errorCallbackURL, err)
		return
	}

	h.handleVerifyMagicLinkSuccess(w, r, result, sessionResult, callbackURL, errorCallbackURL)
}

func (h *magicLinkHandlers) resolveCallbackURLs(state *MagicLinkState) (string, string) {
	callbackURL := ""
	errorCallbackURL := ""
	if state != nil {
		if state.IsNewUser && state.NewUserRedirectURI != "" {
			callbackURL = state.NewUserRedirectURI
		} else {
			callbackURL = state.RedirectURI
		}
		errorCallbackURL = state.ErrorRedirectURI
	}

	return callbackURL, errorCallbackURL
}

func (o *magicLinkHandlers) validateRedirectURLs(data map[string]any) (string, string, string, error) {
	redirectURI, _ := data["redirect_uri"].(string)
	newUserRedirectURI, _ := data["new_user_redirect_uri"].(string)
	errorRedirectURI, _ := data["error_redirect_uri"].(string)

	if redirectURI != "" && !o.httpCore.IsTrustedOrigin(redirectURI) {
		return "", "", "", limen.NewLimenError("redirect_uri is not trusted", http.StatusForbidden, nil)
	}

	if newUserRedirectURI != "" && !o.httpCore.IsTrustedOrigin(newUserRedirectURI) {
		return "", "", "", limen.NewLimenError("new_user_redirect_uri is not trusted", http.StatusForbidden, nil)
	}

	if errorRedirectURI != "" && !o.httpCore.IsTrustedOrigin(errorRedirectURI) {
		return "", "", "", limen.NewLimenError("error_redirect_uri is not trusted", http.StatusForbidden, nil)
	}

	return redirectURI, newUserRedirectURI, errorRedirectURI, nil
}

func (h *magicLinkHandlers) handleVerifyMagicLinkSuccess(w http.ResponseWriter, r *http.Request, result *limen.AuthenticationResult, sessionResult *limen.SessionResult, callbackURL string, errorCallbackURL string) {
	if callbackURL != "" {
		h.responder.RedirectWithSession(w, r, callbackURL, sessionResult)
		return
	}
	h.responder.SessionResponse(w, r, h.plugin.core, result, sessionResult)
}

func (h *magicLinkHandlers) resolveVerifyMagicLinkParams(query url.Values) (string, error) {
	token := query.Get("token")
	if token == "" {
		return "", limen.NewLimenError("token is required", http.StatusBadRequest, nil)
	}
	return token, nil
}

func (h *magicLinkHandlers) handleVerifyMagicLinkError(w http.ResponseWriter, r *http.Request, redirectURI string, errorRedirectURI string, err error) {
	callbackURL := redirectURI
	if errorRedirectURI != "" {
		callbackURL = errorRedirectURI
	}

	if callbackURL != "" {
		parsed, parseErr := url.Parse(callbackURL)
		if parseErr != nil {
			h.responder.Error(w, r, parseErr)
			return
		}
		query := parsed.Query()
		query.Set("error", err.Error())
		parsed.RawQuery = query.Encode()
		h.responder.Redirect(w, r, parsed.String(), http.StatusSeeOther)
		return
	}

	h.responder.Error(w, r, err)
}
