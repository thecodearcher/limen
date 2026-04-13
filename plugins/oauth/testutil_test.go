package oauth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen"
)

// testProvider is a minimal Provider for unit tests that never hits the network.
type testProvider struct {
	name      string
	userInfo  *ProviderUserInfo
	userError error
}

func (p *testProvider) Name() string { return p.name }

func (p *testProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	return &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
		Scopes: []string{"openid", "email"},
	}, nil
}

func (p *testProvider) GetUserInfo(_ context.Context, _ *TokenResponse) (*ProviderUserInfo, error) {
	if p.userError != nil {
		return nil, p.userError
	}
	if p.userInfo != nil {
		return p.userInfo, nil
	}
	return &ProviderUserInfo{
		ID:            "provider-user-123",
		Email:         "oauth@example.com",
		EmailVerified: true,
		Name:          "OAuth User",
	}, nil
}

func newTestOAuthPlugin(t *testing.T, opts ...ConfigOption) (*limen.Limen, *oauthPlugin) {
	t.Helper()

	defaults := []ConfigOption{
		WithProviders(&testProvider{name: "test"}),
	}
	opts = append(defaults, opts...)
	plugin := New(opts...)
	l, _ := limen.NewTestLimen(t, plugin)
	return l, plugin
}

func seedOAuthUser(t *testing.T, l *limen.Limen, email string) *limen.User {
	t.Helper()
	return limen.SeedTestUser(t, l, email)
}

func seedOAuthAccount(t *testing.T, plugin *oauthPlugin, userID any, provider, providerAccountID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	acc := &limen.Account{
		UserID:            userID,
		Provider:          provider,
		ProviderAccountID: providerAccountID,
		AccessToken:       "enc-access",
		RefreshToken:      "enc-refresh",
		Scope:             "openid email",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := plugin.core.Create(ctx, plugin.accountSchema, acc, nil); err != nil {
		t.Fatalf("seedOAuthAccount: %v", err)
	}
}
