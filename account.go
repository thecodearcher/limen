package aegis

import (
	"time"
)

type Account struct {
	ID                   any
	UserID               any
	Provider             string
	ProviderAccountID    string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt *time.Time
	Scope                string
	IDToken              string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	raw                  map[string]any
}

func (a *Account) Raw() map[string]any {
	return a.raw
}
