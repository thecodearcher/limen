package passkey

import (
	"encoding/json"
	"time"

	"github.com/thecodearcher/limen"
)

type Passkey struct {
	ID     any
	UserID any
	Name   *string

	CredentialID string
	PublicKey    string
	SignCount    uint32
	Transports   []string

	AAGUID         string
	BackupEligible bool
	BackupState    bool

	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	raw map[string]any
}

func (p *Passkey) Raw() map[string]any {
	return p.raw
}

const (
	PasskeySchemaTableName limen.SchemaTableName = "passkeys"

	PasskeySchemaUserIDField         limen.SchemaField = "user_id"
	PasskeySchemaNameField           limen.SchemaField = "name"
	PasskeySchemaCredentialIDField   limen.SchemaField = "credential_id" // #nosec G101 -- column name, not a secret
	PasskeySchemaPublicKeyField      limen.SchemaField = "public_key"
	PasskeySchemaSignCountField      limen.SchemaField = "sign_count"
	PasskeySchemaTransportsField     limen.SchemaField = "transports"
	PasskeySchemaAAGUIDField         limen.SchemaField = "aaguid"
	PasskeySchemaBackupEligibleField limen.SchemaField = "backup_eligible"
	PasskeySchemaBackupStateField    limen.SchemaField = "backup_state"
	PasskeySchemaLastUsedAtField     limen.SchemaField = "last_used_at"
)

type passkeySchema struct {
	limen.BaseSchema
}

func newPasskeySchema() *passkeySchema {
	return &passkeySchema{BaseSchema: limen.BaseSchema{
		Serializer: func(data limen.Model) map[string]any {
			passkey := data.(*Passkey)
			return map[string]any{
				"id":              passkey.ID,
				"name":            passkey.Name,
				"credential_id":   passkey.CredentialID,
				"transports":      passkey.Transports,
				"aaguid":          passkey.AAGUID,
				"backup_eligible": passkey.BackupEligible,
				"backup_state":    passkey.BackupState,
				"last_used_at":    passkey.LastUsedAt,
				"created_at":      passkey.CreatedAt.Format(time.RFC3339),
				"updated_at":      passkey.UpdatedAt.Format(time.RFC3339),
			}
		},
	}}
}

func (s *passkeySchema) GetUserIDField() string {
	return s.GetField(PasskeySchemaUserIDField)
}

func (s *passkeySchema) GetCredentialIDField() string {
	return s.GetField(PasskeySchemaCredentialIDField)
}

func (s *passkeySchema) GetPublicKeyField() string {
	return s.GetField(PasskeySchemaPublicKeyField)
}

func (s *passkeySchema) GetSignCountField() string {
	return s.GetField(PasskeySchemaSignCountField)
}

func (s *passkeySchema) GetTransportsField() string {
	return s.GetField(PasskeySchemaTransportsField)
}

func (s *passkeySchema) GetAAGUIDField() string {
	return s.GetField(PasskeySchemaAAGUIDField)
}

func (s *passkeySchema) GetBackupEligibleField() string {
	return s.GetField(PasskeySchemaBackupEligibleField)
}

func (s *passkeySchema) GetBackupStateField() string {
	return s.GetField(PasskeySchemaBackupStateField)
}

func (s *passkeySchema) GetNameField() string {
	return s.GetField(PasskeySchemaNameField)
}

func (s *passkeySchema) GetLastUsedAtField() string {
	return s.GetField(PasskeySchemaLastUsedAtField)
}

func (s *passkeySchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}

func (s *passkeySchema) GetUpdatedAtField() string {
	return s.GetField(limen.SchemaUpdatedAtField)
}

func (s *passkeySchema) ToStorage(data limen.Model) map[string]any {
	passkey := data.(*Passkey)
	payload := map[string]any{
		s.GetUserIDField():         passkey.UserID,
		s.GetNameField():           passkey.Name,
		s.GetCredentialIDField():   passkey.CredentialID,
		s.GetPublicKeyField():      passkey.PublicKey,
		s.GetSignCountField():      passkey.SignCount,
		s.GetAAGUIDField():         passkey.AAGUID,
		s.GetBackupEligibleField(): passkey.BackupEligible,
		s.GetBackupStateField():    passkey.BackupState,
		s.GetLastUsedAtField():     passkey.LastUsedAt,
	}

	if len(passkey.Transports) > 0 {
		if encoded, err := json.Marshal(passkey.Transports); err == nil {
			payload[s.GetTransportsField()] = string(encoded)
		}
	}
	return payload
}

func (s *passkeySchema) FromStorage(data map[string]any) limen.Model {
	return &Passkey{
		ID:           data[s.GetIDField()],
		UserID:       limen.GetValue[any](data[s.GetUserIDField()]),
		Name:         limen.GetNullableValue[string](data[s.GetNameField()]),
		CredentialID: limen.GetValue[string](data[s.GetCredentialIDField()]),
		PublicKey:    limen.GetValue[string](data[s.GetPublicKeyField()]),
		// #nosec G115 -- the counter is only ever stored from a uint32
		SignCount:      uint32(limen.GetValue[int64](data[s.GetSignCountField()])),
		Transports:     limen.ParseJSONFromStorage[[]string](data, s.GetTransportsField()),
		AAGUID:         limen.GetValue[string](data[s.GetAAGUIDField()]),
		BackupEligible: limen.GetValue[bool](data[s.GetBackupEligibleField()]),
		BackupState:    limen.GetValue[bool](data[s.GetBackupStateField()]),
		LastUsedAt:     limen.GetNullableValue[time.Time](data[s.GetLastUsedAtField()]),
		CreatedAt:      limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		UpdatedAt:      limen.GetValue[time.Time](data[s.GetUpdatedAtField()]),
		raw:            data,
	}
}

func buildPasskeyTableDef(schemaConfig *limen.SchemaConfig, schema *passkeySchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(PasskeySchemaTableName),
		PasskeySchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(PasskeySchemaUserIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(PasskeySchemaNameField, limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(PasskeySchemaCredentialIDField, limen.ColumnTypeText),
		limen.WithSchemaField(PasskeySchemaPublicKeyField, limen.ColumnTypeText),
		limen.WithSchemaField(PasskeySchemaSignCountField, limen.ColumnTypeInt64, limen.WithDefaultValue("0")),
		limen.WithSchemaField(PasskeySchemaTransportsField, limen.ColumnTypeText, limen.WithNullable(true)),
		limen.WithSchemaField(PasskeySchemaAAGUIDField, limen.ColumnTypeString),
		limen.WithSchemaField(PasskeySchemaBackupEligibleField, limen.ColumnTypeBool, limen.WithDefaultValue("false")),
		limen.WithSchemaField(PasskeySchemaBackupStateField, limen.ColumnTypeBool, limen.WithDefaultValue("false")),
		limen.WithSchemaField(PasskeySchemaLastUsedAtField, limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_passkeys_credential_id", []limen.SchemaField{PasskeySchemaCredentialIDField}),
		limen.WithSchemaIndex("idx_passkeys_user_id", []limen.SchemaField{PasskeySchemaUserIDField}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_passkeys_user",
			Column:           PasskeySchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}
