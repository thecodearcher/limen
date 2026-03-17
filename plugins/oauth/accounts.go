package oauth

import (
	"context"

	"github.com/thecodearcher/limen"
)

// ListAccountsForUser returns all OAuth accounts linked to a user.
func (o *oauthPlugin) ListAccountsForUser(ctx context.Context, userID any) ([]*limen.Account, error) {
	models, err := o.core.FindMany(ctx, o.accountSchema, []limen.Where{
		limen.Eq(o.accountSchema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}

	accounts := make([]*limen.Account, 0, len(models))
	for _, model := range models {
		accounts = append(accounts, model.(*limen.Account))
	}
	return accounts, nil
}

func (o *oauthPlugin) UnlinkAccount(ctx context.Context, user *limen.User, providerName string) error {
	accountCount, err := o.core.Count(ctx, o.accountSchema, []limen.Where{
		limen.Eq(o.accountSchema.GetUserIDField(), user.ID),
	})

	if err != nil {
		return err
	}

	if accountCount == 0 {
		return ErrAccountNotFound
	}

	if user.Password == nil && accountCount == 1 {
		return ErrCannotUnlinkOnlyAccount
	}

	return o.core.Delete(ctx, o.accountSchema, []limen.Where{
		limen.Eq(o.accountSchema.GetUserIDField(), user.ID),
		limen.Eq(o.accountSchema.GetProviderField(), providerName),
	})
}
