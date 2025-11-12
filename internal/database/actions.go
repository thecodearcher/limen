package database

import (
	"context"
	"errors"
	"time"

	"sync"

	"github.com/thecodearcher/aegis"
)

var (
	databaseActions *DatabaseActionHelper
	once            sync.Once
)

type DatabaseActionHelper struct {
	core *aegis.AegisCore
}

type InternalDatabaseActionsOption func(i *DatabaseActionHelper)

func NewCommonDatabaseActionsHelper(core *aegis.AegisCore) *DatabaseActionHelper {
	once.Do(func() {
		i := &DatabaseActionHelper{core: core}
		databaseActions = i
	})
	return databaseActions
}

func (i *DatabaseActionHelper) FindUserByEmail(ctx context.Context, email string) (*aegis.User, error) {
	return FindOne(ctx, i.core, &i.core.Schema.User, []aegis.Where{
		aegis.Eq(i.core.Schema.User.GetEmailField(), email),
	}, nil)
}

func (i *DatabaseActionHelper) CreateUser(ctx context.Context, data *aegis.User, additionalFields map[string]any) error {
	if err := Create(ctx, i.core, &i.core.Schema.User, data, additionalFields); err != nil {
		return err
	}

	return nil
}

func (i *DatabaseActionHelper) CreateVerification(ctx context.Context, action string, identifier string, token string, expiresAt time.Duration) (*aegis.Verification, error) {
	if identifier == "" {
		return nil, errors.New("identifier is required")
	}
	verificationSchema := i.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)
	if err := Create(ctx, i.core, &verificationSchema, &aegis.Verification{
		Subject:   actionValue,
		Value:     token,
		ExpiresAt: time.Now().Add(expiresAt).UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil); err != nil {
		return nil, err
	}

	return i.FindVerificationByAction(ctx, action, identifier)
}

func (i *DatabaseActionHelper) FindVerificationByAction(ctx context.Context, action string, identifier string) (*aegis.Verification, error) {
	verificationSchema := i.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)
	return FindOne(ctx, i.core, &verificationSchema,
		[]aegis.Where{
			aegis.Eq(verificationSchema.GetSubjectField(), actionValue),
		},
		[]aegis.OrderBy{
			{
				Column:    verificationSchema.GetCreatedAtField(),
				Direction: aegis.OrderByDesc,
			},
		})
}

func (i *DatabaseActionHelper) FindValidVerificationByToken(ctx context.Context, token string) (*aegis.Verification, error) {
	verificationSchema := i.core.Schema.Verification
	return FindOne(ctx, i.core, &verificationSchema,
		[]aegis.Where{
			aegis.Eq(verificationSchema.GetValueField(), token),
			aegis.Gt(verificationSchema.GetExpiresAtField(), time.Now().UTC()),
		},
		[]aegis.OrderBy{
			{
				Column:    verificationSchema.GetCreatedAtField(),
				Direction: aegis.OrderByDesc,
			},
		})
}

func (i *DatabaseActionHelper) DeleteExpiredVerifications(ctx context.Context) error {
	verificationSchema := i.core.Schema.Verification
	return i.core.DB.Delete(ctx, verificationSchema.GetTableName(), []aegis.Where{
		aegis.Lt(verificationSchema.GetExpiresAtField(), time.Now().UTC()),
	})
}

func (i *DatabaseActionHelper) RevokeVerification(ctx context.Context, token string) error {
	verificationSchema := i.core.Schema.Verification
	return i.core.DB.Update(ctx, verificationSchema.GetTableName(), []aegis.Where{
		aegis.Eq(verificationSchema.GetValueField(), token),
	}, map[string]any{
		verificationSchema.GetExpiresAtField(): time.Now().Add(-time.Minute).UTC(),
	})
}

func (i *DatabaseActionHelper) UpdateUser(ctx context.Context, data *aegis.User, conditions []aegis.Where) error {
	if err := Update(ctx, i.core, &i.core.Schema.User, data, conditions); err != nil {
		return err
	}
	return nil
}
