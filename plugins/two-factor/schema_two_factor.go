package twofactor

import (
	"time"

	"github.com/thecodearcher/limen"
)

type TwoFactor struct {
	ID          any
	UserID      any
	Secret      string
	BackupCodes string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	raw         map[string]any
}

func (t *TwoFactor) Raw() map[string]any {
	return t.raw
}

type twoFactorSchema struct {
	limen.BaseSchema
}

type SchemaConfigTwoFactorOption func(*twoFactorSchema)

func newDefaultTwoFactorSchema(opts ...SchemaConfigTwoFactorOption) *twoFactorSchema {
	schema := &twoFactorSchema{
		BaseSchema: limen.BaseSchema{},
	}

	for _, opt := range opts {
		opt(schema)
	}

	return schema
}

func (t *twoFactorSchema) GetUserIDField() string {
	return t.GetField(TwoFactorSchemaUserIDField)
}

func (t *twoFactorSchema) GetSecretField() string {
	return t.GetField(TwoFactorSchemaSecretField)
}

func (t *twoFactorSchema) GetBackupCodesField() string {
	return t.GetField(TwoFactorSchemaBackupCodesField)
}

func (t *twoFactorSchema) GetCreatedAtField() string {
	return t.GetField(limen.SchemaCreatedAtField)
}

func (t *twoFactorSchema) GetUpdatedAtField() string {
	return t.GetField(limen.SchemaUpdatedAtField)
}

func (t *twoFactorSchema) ToStorage(data limen.Model) map[string]any {
	twoFactor := data.(*TwoFactor)
	return map[string]any{
		t.GetUserIDField():      twoFactor.UserID,
		t.GetSecretField():      twoFactor.Secret,
		t.GetBackupCodesField(): twoFactor.BackupCodes,
	}
}

func (t *twoFactorSchema) FromStorage(data map[string]any) limen.Model {
	return &TwoFactor{
		ID:          data[t.GetIDField()],
		UserID:      data[t.GetUserIDField()],
		Secret:      data[t.GetSecretField()].(string),
		BackupCodes: data[t.GetBackupCodesField()].(string),
		raw:         data,
	}
}
