package oauth

import (
	"context"

	"github.com/thecodearcher/aegis"
)

// ListAccountsForUser returns all OAuth accounts linked to a user.
func (o *oauthFeature) ListAccountsForUser(ctx context.Context, userID any) ([]*aegis.Account, error) {
	models, err := o.core.FindMany(ctx, o.accountSchema, []aegis.Where{
		aegis.Eq(o.accountSchema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}

	accounts := make([]*aegis.Account, 0, len(models))
	for _, model := range models {
		accounts = append(accounts, model.(*aegis.Account))
	}
	return accounts, nil
}
