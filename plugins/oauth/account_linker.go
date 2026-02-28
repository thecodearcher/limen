package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/thecodearcher/aegis"
)

func (o *oauthPlugin) CreateOrLinkAccount(ctx context.Context, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error) {
	if err := o.validateProviderInfo(info); err != nil {
		return nil, err
	}

	existingAccount, err := o.core.DBAction.FindAccountByProviderAndProviderID(ctx, info.Provider, info.ProviderAccountID)
	if err != nil && !errors.Is(err, aegis.ErrRecordNotFound) {
		return nil, err
	}

	if existingAccount != nil {
		return o.updateExistingAccount(ctx, existingAccount, info)
	}

	user, err := o.findUserByEmail(ctx, info.Email)
	if err != nil {
		return nil, err
	}

	if user != nil {
		return o.linkAccountToUser(ctx, user, info)
	}

	if !o.config.requireExplicitSignUp {
		return o.createUserAndLinkAccount(ctx, info)
	}

	return nil, ErrAccountNotFound
}

// LinkAccountToCurrentUser links an OAuth provider account to an already-authenticated user.
// If the provider account is already linked to the same user, tokens are updated.
// If the provider account is linked to a different user, an error is returned.
func (o *oauthPlugin) LinkAccountToCurrentUser(ctx context.Context, user *aegis.User, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error) {
	if err := o.validateProviderInfo(info); err != nil {
		return nil, err
	}

	existingAccount, err := o.core.DBAction.FindAccountByProviderAndProviderID(ctx, info.Provider, info.ProviderAccountID)
	if err != nil && !errors.Is(err, aegis.ErrRecordNotFound) {
		return nil, err
	}

	if existingAccount != nil {
		if existingAccount.UserID != user.ID {
			return nil, ErrAccountAlreadyLinkedToAnotherUser
		}
		return o.updateExistingAccount(ctx, existingAccount, info)
	}

	if !o.config.allowLinkingDifferentEmails && info.Email != user.Email {
		return nil, ErrAccountCannotBeLinkedToDifferentEmail
	}

	return o.linkAccountToUser(ctx, user, info)
}

func (o *oauthPlugin) validateProviderInfo(info *aegis.OAuthAccountProfile) error {
	if info == nil {
		return aegis.NewAegisError("info is required", http.StatusBadRequest, nil)
	}

	if info.Provider == "" || info.ProviderAccountID == "" {
		return aegis.NewAegisError("provider and provider_account_id are required", http.StatusBadRequest, nil)
	}

	if info.Email == "" {
		return aegis.NewAegisError("email is required", http.StatusBadRequest, nil)
	}

	return nil
}

func (o *oauthPlugin) updateExistingAccount(ctx context.Context, account *aegis.Account, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error) {
	tokens, err := o.encryptTokens(info)
	if err != nil {
		return nil, err
	}
	updated := &aegis.Account{
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		Scope:                info.Scope,
		AccessTokenExpiresAt: info.AccessTokenExpiresAt,
		IDToken:              tokens.IDToken,
		UpdatedAt:            time.Now(),
	}
	if err := o.core.Update(ctx, o.accountSchema, updated, []aegis.Where{
		aegis.Eq(o.accountSchema.GetIDField(), account.ID),
	}); err != nil {
		return nil, err
	}
	user, err := o.core.DBAction.FindUserByID(ctx, account.UserID)
	if err != nil {
		return nil, err
	}
	return &aegis.AuthenticationResult{User: user}, nil
}

func (o *oauthPlugin) linkAccountToUser(ctx context.Context, user *aegis.User, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error) {
	tokens, err := o.encryptTokens(info)
	if err != nil {
		return nil, err
	}
	acc := newAccountFromOAuthProfile(user.ID, info, tokens)
	if err := o.core.Create(ctx, o.accountSchema, acc, nil); err != nil {
		return nil, err
	}
	return &aegis.AuthenticationResult{User: user}, nil
}

func (o *oauthPlugin) createUserAndLinkAccount(ctx context.Context, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error) {
	user := &aegis.User{
		Email:           info.Email,
		EmailVerifiedAt: nil,
	}
	if info.EmailVerified {
		now := time.Now()
		user.EmailVerifiedAt = &now
	}

	additional := map[string]any{}
	if o.config.mapProfileToUser != nil {
		additional = o.config.mapProfileToUser(info)
	}

	tokens, err := o.encryptTokens(info)
	if err != nil {
		return nil, err
	}

	var linkedUser *aegis.User
	err = o.core.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := o.core.DBAction.CreateUser(txCtx, user, additional); err != nil {
			return err
		}

		linkedUser, err = o.core.DBAction.FindUserByEmail(txCtx, info.Email)
		if err != nil {
			return err
		}

		acc := newAccountFromOAuthProfile(linkedUser.ID, info, tokens)
		return o.core.Create(txCtx, o.accountSchema, acc, nil)
	})
	if err != nil {
		return nil, err
	}
	return &aegis.AuthenticationResult{User: linkedUser}, nil
}

func (o *oauthPlugin) findUserByEmail(ctx context.Context, email string) (*aegis.User, error) {
	user, err := o.core.DBAction.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, aegis.ErrRecordNotFound) {
		return nil, err
	}
	return user, nil
}

func (o *oauthPlugin) findAccountByUserIDAndProvider(ctx context.Context, userID any, providerName string) (*aegis.Account, error) {
	raw, err := o.core.FindOne(ctx, o.accountSchema, []aegis.Where{
		aegis.Eq(o.accountSchema.GetUserIDField(), userID),
		aegis.Eq(o.accountSchema.GetProviderField(), providerName),
	}, nil)
	if err != nil {
		return nil, err
	}
	return raw.(*aegis.Account), nil
}

func (o *oauthPlugin) encryptToken(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	cipher, err := aegis.EncryptXChaCha(plain, o.config.secret, nil)
	if err != nil {
		return "", fmt.Errorf("oauth: failed to encrypt token: %w", err)
	}
	return cipher, nil
}

func (o *oauthPlugin) decryptToken(cipher string) (string, error) {
	if cipher == "" {
		return "", nil
	}
	plain, err := aegis.DecryptXChaCha(cipher, o.config.secret, nil)
	if err != nil {
		return "", fmt.Errorf("oauth: failed to decrypt token: %w", err)
	}
	fmt.Printf("decryptToken plain: %s\n", plain)
	return plain, nil
}

func (o *oauthPlugin) encryptTokens(info *aegis.OAuthAccountProfile) (*OAuthTokens, error) {
	if o.config.disableTokensEncryption {
		return &OAuthTokens{
			AccessToken:  info.AccessToken,
			RefreshToken: info.RefreshToken,
			IDToken:      info.IDToken,
		}, nil
	}

	if o.config.encryptTokens != nil {
		return o.config.encryptTokens(o.config.secret, &OAuthTokens{
			AccessToken:  info.AccessToken,
			RefreshToken: info.RefreshToken,
			IDToken:      info.IDToken,
		})
	}
	access, err := o.encryptToken(info.AccessToken)
	if err != nil {
		return nil, err
	}
	refresh, err := o.encryptToken(info.RefreshToken)
	if err != nil {
		return nil, err
	}
	idToken, err := o.encryptToken(info.IDToken)
	if err != nil {
		return nil, err
	}
	return &OAuthTokens{AccessToken: access, RefreshToken: refresh, IDToken: idToken}, nil
}

func (o *oauthPlugin) decryptTokens(account *aegis.Account) (*OAuthTokens, error) {
	if o.config.disableTokensEncryption {
		return &OAuthTokens{
			AccessToken:  account.AccessToken,
			RefreshToken: account.RefreshToken,
			IDToken:      account.IDToken,
		}, nil
	}

	if o.config.decryptTokens != nil {
		return o.config.decryptTokens(o.config.secret, &OAuthTokens{
			AccessToken:  account.AccessToken,
			RefreshToken: account.RefreshToken,
			IDToken:      account.IDToken,
		})
	}

	access, err := o.decryptToken(account.AccessToken)
	if err != nil {
		return nil, err
	}

	refresh, err := o.decryptToken(account.RefreshToken)
	if err != nil {
		return nil, err
	}

	idToken, err := o.decryptToken(account.IDToken)
	if err != nil {
		return nil, err
	}

	return &OAuthTokens{AccessToken: access, RefreshToken: refresh, IDToken: idToken}, nil
}
