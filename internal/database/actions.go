package database

import (
	"context"
	"fmt"
	"time"

	"sync"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/schemas"
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

func (i *DatabaseActionHelper) CreateUser(ctx context.Context, data schemas.User, additionalFields map[string]any) error {
	if err := Create[schemas.User](ctx, i.core, &i.core.Schema.User, data, additionalFields); err != nil {
		return err
	}
	return nil
}

func (i *DatabaseActionHelper) CreateVerification(ctx context.Context, action string, identifier string, token string, expiresAt time.Duration) (*schemas.Verification, error) {
	fmt.Printf("expiresAt: %v\n", i)
	verificationSchema := i.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)
	if err := Create[schemas.Verification](ctx, i.core, &verificationSchema, schemas.Verification{
		Subject:   actionValue,
		Value:     token,
		ExpiresAt: time.Now().Add(expiresAt).UTC(),
	}, nil); err != nil {
		return nil, err
	}

	return i.FindVerificationByAction(ctx, action, identifier)
}

func (i *DatabaseActionHelper) FindVerificationByAction(ctx context.Context, action string, identifier string) (*schemas.Verification, error) {
	verificationSchema := i.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)
	return FindOne[schemas.Verification](ctx, i.core.DB, &verificationSchema,
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

func (i *DatabaseActionHelper) FindVerificationByToken(ctx context.Context, token string) (*schemas.Verification, error) {
	verificationSchema := i.core.Schema.Verification
	return FindOne[schemas.Verification](ctx, i.core.DB, &verificationSchema,
		[]aegis.Where{
			aegis.Eq(verificationSchema.GetValueField(), token),
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
		aegis.Lt(verificationSchema.GetExpiresAtField(), time.Now()),
	})
}

func (i *DatabaseActionHelper) UpdateUser(ctx context.Context, data schemas.User, conditions []aegis.Where) error {
	if err := Update[schemas.User](ctx, i.core.DB, &i.core.Schema.User, data, conditions); err != nil {
		return err
	}
	return nil
}
