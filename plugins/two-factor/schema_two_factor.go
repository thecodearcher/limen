package twofactor

import (
	"time"

	"github.com/thecodearcher/aegis"
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
	aegis.BaseSchema
}

type SchemaConfigTwoFactorOption func(*twoFactorSchema)

func newDefaultTwoFactorSchema(opts ...SchemaConfigTwoFactorOption) *twoFactorSchema {
	schema := &twoFactorSchema{
		BaseSchema: aegis.BaseSchema{},
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
	return t.GetField(aegis.SchemaCreatedAtField)
}

func (t *twoFactorSchema) GetUpdatedAtField() string {
	return t.GetField(aegis.SchemaUpdatedAtField)
}

func (t *twoFactorSchema) ToStorage(data aegis.Model) map[string]any {
	twoFactor := data.(*TwoFactor)
	return map[string]any{
		t.GetUserIDField():      twoFactor.UserID,
		t.GetSecretField():      twoFactor.Secret,
		t.GetBackupCodesField(): twoFactor.BackupCodes,
	}
}

func (t *twoFactorSchema) FromStorage(data map[string]any) aegis.Model {
	return &TwoFactor{
		ID:          data[t.GetIDField()],
		UserID:      data[t.GetUserIDField()],
		Secret:      data[t.GetSecretField()].(string),
		BackupCodes: data[t.GetBackupCodesField()].(string),
		raw:         data,
	}
}
