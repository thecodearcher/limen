package passkey

import (
	"time"

	"github.com/thecodearcher/limen"
)

// PasskeyRecord is the internal storage row for a single registered
// credential. The CredentialID and PublicKey are stored base64url-encoded
// so they can live in string columns.
type PasskeyRecord struct {
	ID                any
	UserID            any
	Name              string
	CredentialIDBase64 string
	PublicKeyBase64   string
	Counter           int64
	DeviceType        string
	BackedUp          bool
	Transports        string
	AAGUID            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	raw               map[string]any
}

func (p *PasskeyRecord) Raw() map[string]any { return p.raw }

type passkeySchema struct {
	limen.BaseSchema
}

func newPasskeySchema() *passkeySchema {
	return &passkeySchema{BaseSchema: limen.BaseSchema{}}
}

func (s *passkeySchema) GetUserIDField() string {
	return s.GetField(PasskeySchemaUserIDField)
}
func (s *passkeySchema) GetNameField() string {
	return s.GetField(PasskeySchemaNameField)
}
func (s *passkeySchema) GetCredentialIDField() string {
	return s.GetField(PasskeySchemaCredentialIDField)
}
func (s *passkeySchema) GetPublicKeyField() string {
	return s.GetField(PasskeySchemaPublicKeyField)
}
func (s *passkeySchema) GetCounterField() string {
	return s.GetField(PasskeySchemaCounterField)
}
func (s *passkeySchema) GetDeviceTypeField() string {
	return s.GetField(PasskeySchemaDeviceTypeField)
}
func (s *passkeySchema) GetBackedUpField() string {
	return s.GetField(PasskeySchemaBackedUpField)
}
func (s *passkeySchema) GetTransportsField() string {
	return s.GetField(PasskeySchemaTransportsField)
}
func (s *passkeySchema) GetAAGUIDField() string {
	return s.GetField(PasskeySchemaAAGUIDField)
}
func (s *passkeySchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}
func (s *passkeySchema) GetUpdatedAtField() string {
	return s.GetField(limen.SchemaUpdatedAtField)
}

func (s *passkeySchema) ToStorage(data limen.Model) map[string]any {
	rec := data.(*PasskeyRecord)
	return map[string]any{
		s.GetUserIDField():       rec.UserID,
		s.GetNameField():         rec.Name,
		s.GetCredentialIDField(): rec.CredentialIDBase64,
		s.GetPublicKeyField():    rec.PublicKeyBase64,
		s.GetCounterField():      rec.Counter,
		s.GetDeviceTypeField():   rec.DeviceType,
		s.GetBackedUpField():     rec.BackedUp,
		s.GetTransportsField():   rec.Transports,
		s.GetAAGUIDField():       rec.AAGUID,
	}
}

func (s *passkeySchema) FromStorage(data map[string]any) limen.Model {
	rec := &PasskeyRecord{
		ID:                 data[s.GetIDField()],
		UserID:             data[s.GetUserIDField()],
		CredentialIDBase64: stringOr(data[s.GetCredentialIDField()], ""),
		PublicKeyBase64:    stringOr(data[s.GetPublicKeyField()], ""),
		Counter:            int64Or(data[s.GetCounterField()], 0),
		DeviceType:         stringOr(data[s.GetDeviceTypeField()], ""),
		BackedUp:           boolOr(data[s.GetBackedUpField()], false),
		Transports:         stringOr(data[s.GetTransportsField()], ""),
		AAGUID:             stringOr(data[s.GetAAGUIDField()], ""),
		Name:               stringOr(data[s.GetNameField()], ""),
		raw:                data,
	}
	if v, ok := data[s.GetCreatedAtField()].(time.Time); ok {
		rec.CreatedAt = v
	}
	if v, ok := data[s.GetUpdatedAtField()].(time.Time); ok {
		rec.UpdatedAt = v
	}
	return rec
}

// stringOr / int64Or / boolOr defend against drivers that emit different
// concrete types for the same logical column (e.g. lib/pq returning
// []byte instead of string for TEXT, or int64 vs int for counters).
func stringOr(v any, fallback string) string {
	switch t := v.(type) {
	case nil:
		return fallback
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fallback
	}
}

func int64Or(v any, fallback int64) int64 {
	switch t := v.(type) {
	case nil:
		return fallback
	case int64:
		return t
	case int:
		return int64(t)
	case int32:
		return int64(t)
	default:
		return fallback
	}
}

func boolOr(v any, fallback bool) bool {
	switch t := v.(type) {
	case nil:
		return fallback
	case bool:
		return t
	default:
		return fallback
	}
}
