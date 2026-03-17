package oauth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen"
)

func (o *oauthPlugin) getProviderConfig(provider Provider) (*oauth2.Config, []oauth2.AuthCodeOption) {
	defaultOpts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("response_type", "code"),
	}

	config, authOpts := provider.OAuth2Config()
	config.RedirectURL = o.constructProviderRedirectURL(provider, config)
	opts := append(defaultOpts, authOpts...)
	return config, opts
}

func (o *oauthPlugin) constructProviderRedirectURL(provider Provider, config *oauth2.Config) string {
	if config.RedirectURL != "" {
		return config.RedirectURL
	}
	return o.core.GetBaseURLWithPluginPath(limen.PluginOAuth, fmt.Sprintf("%s/callback", provider.Name()))
}

func (o *oauthPlugin) buildAuthorizationURL(ctx context.Context, provider Provider, stateToken, verifier string) (string, error) {
	if pkce, ok := provider.(PKCEEnabledProvider); ok && !pkce.PKCEEnabled() {
		verifier = ""
	}
	config, authOpts := o.getProviderConfig(provider)
	if builder, ok := provider.(AuthorizationURLBuilder); ok {
		return builder.BuildAuthorizationURL(ctx, stateToken, verifier, config.RedirectURL)
	}
	return BuildAuthCodeURL(config, stateToken, verifier, authOpts...), nil
}

func (o *oauthPlugin) exchangeCodeForTokens(ctx context.Context, provider Provider, code, codeVerifier string) (*TokenResponse, error) {
	config, _ := o.getProviderConfig(provider)
	if exchanger, ok := provider.(TokenExchanger); ok {
		return exchanger.ExchangeAuthorizationCode(ctx, code, codeVerifier, config.RedirectURL)
	}
	if pkce, ok := provider.(PKCEEnabledProvider); ok && !pkce.PKCEEnabled() {
		codeVerifier = ""
	}
	return ExchangeCode(ctx, config, code, codeVerifier)
}

func (o *oauthPlugin) validateRedirectURLs(request *OAuthAuthorizeURLData) (string, string, error) {
	redirectURI := request.RedirectURI
	if redirectURI == "" {
		redirectURI = o.core.GetBaseURL()
	}

	if !o.httpCore.IsTrustedOrigin(redirectURI) {
		return "", "", limen.NewLimenError("redirect_uri is not trusted", http.StatusForbidden, nil)
	}

	if request.ErrorRedirectURI != "" && !o.httpCore.IsTrustedOrigin(request.ErrorRedirectURI) {
		return "", "", limen.NewLimenError("error_redirect_uri is not trusted", http.StatusForbidden, nil)
	}

	return redirectURI, request.ErrorRedirectURI, nil
}

func (o *oauthPlugin) resolveStateData(ctx context.Context, state, cookieNonce string) (map[string]any, error) {
	if state == "" || cookieNonce == "" {
		return nil, limen.NewLimenError("state and cookie nonce are required", http.StatusBadRequest, nil)
	}
	return o.stateStore.Validate(ctx, state, cookieNonce)
}

// GetAuthorizationURL returns the provider's authorization URL along with the cookie value.
func (o *oauthPlugin) GetAuthorizationURL(ctx context.Context, providerName string, request *OAuthAuthorizeURLData) (string, string, error) {
	provider, ok := o.providers[providerName]
	if !ok {
		return "", "", ErrProviderNotFound
	}

	redirectURI, errorRedirectURI, err := o.validateRedirectURLs(request)
	if err != nil {
		return "", "", err
	}

	verifier := generateCodeVerifier()

	data := map[string]any{
		pkceDataKey:         verifier,
		additionalDataKey:   request.AdditionalData,
		redirectURIKey:      redirectURI,
		errorRedirectURIKey: errorRedirectURI,
	}

	stateToken, cookieValue, err := o.stateStore.Generate(ctx, data)
	if err != nil {
		return "", "", err
	}

	url, err := o.buildAuthorizationURL(ctx, provider, stateToken, verifier)
	if err != nil {
		return "", "", err
	}
	return url, cookieValue, nil
}

// ExchangeAuthorizationCodeForTokens exchanges the authorization code for tokens.
func (o *oauthPlugin) ExchangeAuthorizationCodeForTokens(ctx context.Context, provider Provider, stateData map[string]any, code string) (*TokenResponse, error) {
	codeVerifier, ok := stateData[pkceDataKey].(string)
	if !ok {
		return nil, ErrPKCEVerifierNotFound
	}

	token, err := o.exchangeCodeForTokens(ctx, provider, code, codeVerifier)
	if err != nil {
		return nil, limen.NewLimenError(err.Error(), http.StatusBadRequest, err)
	}

	return token, nil
}

// GetUserInfoWithTokens fetches the user info from the provider using the access token.
func (o *oauthPlugin) GetUserInfoWithTokens(ctx context.Context, provider Provider, token *TokenResponse) (*limen.OAuthAccountProfile, error) {
	var userInfo *ProviderUserInfo
	var err error

	if o.config.getUserInfo != nil {
		userInfo, err = o.config.getUserInfo(ctx, provider.Name(), token)
	} else {
		userInfo, err = provider.GetUserInfo(ctx, token)
	}

	if err != nil {
		return nil, err
	}

	if userInfo == nil {
		return nil, limen.NewLimenError("provider user info not found", http.StatusUnauthorized, nil)
	}

	if userInfo.Email == "" || userInfo.ID == "" {
		return nil, limen.NewLimenError("provider user email or id not found", http.StatusBadRequest, nil)
	}

	var expiresAt *time.Time
	if !token.ExpiresAt.IsZero() {
		expiresAt = &token.ExpiresAt
	}

	return &limen.OAuthAccountProfile{
		Provider:             provider.Name(),
		ProviderAccountID:    userInfo.ID,
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpiresAt: expiresAt,
		Scope:                token.Scope,
		IDToken:              token.IDToken,
		Email:                userInfo.Email,
		EmailVerified:        userInfo.EmailVerified,
		Name:                 userInfo.Name,
		AvatarURL:            userInfo.AvatarURL,
		Raw:                  userInfo.Raw,
	}, nil
}

func (o *oauthPlugin) HandleOAuthCallback(ctx context.Context, providerName, code, state, cookieNonce string, callbackErr *CallbackError) (*limen.OAuthAccountProfile, map[string]any, error) {
	provider, ok := o.providers[providerName]
	if !ok {
		return nil, nil, ErrProviderNotFound
	}

	stateData, err := o.resolveStateData(ctx, state, cookieNonce)
	if err != nil {
		return nil, nil, err
	}

	if callbackErr != nil {
		return nil, stateData, callbackErr.ToLimenError()
	}

	if code == "" {
		return nil, stateData, limen.NewLimenError("authorization code is required", http.StatusBadRequest, nil)
	}

	token, err := o.ExchangeAuthorizationCodeForTokens(ctx, provider, stateData, code)
	if err != nil {
		return nil, stateData, err
	}

	userInfo, err := o.GetUserInfoWithTokens(ctx, provider, token)
	if err != nil {
		return nil, stateData, err
	}

	return userInfo, stateData, nil
}

// AuthenticateWithProvider runs the full OAuth callback flow
func (o *oauthPlugin) AuthenticateWithProvider(ctx context.Context, providerName, code, state, cookieNonce string, callbackErr *CallbackError) (*limen.AuthenticationResult, map[string]any, error) {
	userInfo, stateData, err := o.HandleOAuthCallback(ctx, providerName, code, state, cookieNonce, callbackErr)
	if err != nil {
		return nil, stateData, err
	}

	additionalData, ok := stateData[additionalDataKey].(map[string]any)
	if !ok || additionalData == nil {
		additionalData = make(map[string]any)
	}

	if additionalData[linkUserIdKey] != nil {
		userId := additionalData[linkUserIdKey]
		user, err := o.core.DBAction.FindUserByID(ctx, userId)
		if err != nil {
			return nil, stateData, err
		}

		result, err := o.LinkAccountToCurrentUser(ctx, user, userInfo)
		if err != nil {
			return nil, stateData, err
		}

		return result, stateData, nil
	}

	result, err := o.CreateOrLinkAccount(ctx, userInfo)
	if err != nil {
		return nil, stateData, err
	}

	return result, stateData, nil
}
