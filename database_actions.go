package aegis

import (
	"context"
	"errors"
	"time"
)

// DatabaseActionHelper provides common database operations for plugins.
type DatabaseActionHelper struct {
	core *AegisCore
}

func newCommonDatabaseActionsHelper(core *AegisCore) *DatabaseActionHelper {
	return &DatabaseActionHelper{core: core}
}

func (i *DatabaseActionHelper) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return FindOne(ctx, i.core, i.core.Schema.User, []Where{
		Eq(i.core.Schema.User.GetEmailField(), email),
	}, nil)
}

func (i *DatabaseActionHelper) FindUser(ctx context.Context, conditions []Where) (*User, error) {
	return FindOne(ctx, i.core, i.core.Schema.User, conditions, nil)
}

func (i *DatabaseActionHelper) FindUserByID(ctx context.Context, id any) (*User, error) {
	return FindOne(ctx, i.core, i.core.Schema.User, []Where{
		Eq(i.core.Schema.User.GetIDField(), id),
	}, nil)
}

func (i *DatabaseActionHelper) CreateUser(ctx context.Context, data *User, additionalFields map[string]any) error {
	if err := Create(ctx, i.core, i.core.Schema.User, data, additionalFields); err != nil {
		return err
	}

	return nil
}

func (i *DatabaseActionHelper) UpdateUser(ctx context.Context, updatedUser *User, conditions []Where) error {
	if err := Update(ctx, i.core, i.core.Schema.User, updatedUser, conditions); err != nil {
		return err
	}
	return nil
}

func (i *DatabaseActionHelper) CreateVerification(ctx context.Context, action string, identifier string, token string, expiresAt time.Duration) (*Verification, error) {
	if identifier == "" {
		return nil, errors.New("identifier is required")
	}
	verificationSchema := i.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)

	if err := Create(ctx, i.core, verificationSchema, &Verification{
		Subject:   actionValue,
		Value:     token,
		ExpiresAt: time.Now().Add(expiresAt),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil); err != nil {
		return nil, err
	}

	return i.FindVerificationByAction(ctx, action, identifier)
}

func (i *DatabaseActionHelper) FindVerificationByAction(ctx context.Context, action string, identifier string) (*Verification, error) {
	verificationSchema := i.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)
	return FindOne(ctx, i.core, verificationSchema,
		[]Where{
			Eq(verificationSchema.GetSubjectField(), actionValue),
		},
		[]OrderBy{
			{
				Column:    verificationSchema.GetCreatedAtField(),
				Direction: OrderByDesc,
			},
		})
}

func (i *DatabaseActionHelper) FindValidVerificationByToken(ctx context.Context, token string) (*Verification, error) {
	verificationSchema := i.core.Schema.Verification
	return FindOne(ctx, i.core, verificationSchema,
		[]Where{
			Eq(verificationSchema.GetValueField(), token),
			Gt(verificationSchema.GetExpiresAtField(), time.Now()),
		},
		[]OrderBy{
			{
				Column:    verificationSchema.GetCreatedAtField(),
				Direction: OrderByDesc,
			},
		})
}

func (i *DatabaseActionHelper) DeleteVerificationToken(ctx context.Context, token string) error {
	verificationSchema := i.core.Schema.Verification
	return Delete(ctx, i.core, verificationSchema, []Where{
		Eq(verificationSchema.GetValueField(), token),
	})
}

func (i *DatabaseActionHelper) CreateSession(ctx context.Context, data *Session, additionalFields map[string]any) error {
	if err := Create(ctx, i.core, i.core.Schema.Session, data, additionalFields); err != nil {
		return err
	}
	return nil
}

func (i *DatabaseActionHelper) UpdateSession(ctx context.Context, data *Session, conditions []Where) error {
	if err := Update(ctx, i.core, i.core.Schema.Session, data, conditions); err != nil {
		return err
	}
	return nil
}

func (i *DatabaseActionHelper) FindSessionByToken(ctx context.Context, sessionToken string) (*Session, error) {
	sessionSchema := i.core.Schema.Session
	return FindOne(ctx, i.core, sessionSchema, []Where{
		Eq(sessionSchema.GetTokenField(), sessionToken),
	}, nil)
}

func (i *DatabaseActionHelper) DeleteSessionByToken(ctx context.Context, sessionToken string) error {
	sessionSchema := i.core.Schema.Session
	return Delete(ctx, i.core, sessionSchema, []Where{
		Eq(sessionSchema.GetTokenField(), sessionToken),
	})
}

func (i *DatabaseActionHelper) DeleteSessionByUserID(ctx context.Context, userID any) error {
	sessionSchema := i.core.Schema.Session
	return Delete(ctx, i.core, sessionSchema, []Where{
		Eq(sessionSchema.GetUserIDField(), userID),
	})
}
