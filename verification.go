package aegis

import "time"

type Verification struct {
	Subject   string
	Value     string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	raw       map[string]any
}

func (v Verification) Raw() map[string]any {
	return v.raw
}

type VerificationSchema struct {
	// name of the table in the database
	TableName TableName
	// mapping of the verification schema to the database columns
	Fields VerificationFields
	// A function to return a map of additional fields to be added to the schema when creating a record
	AdditionalFields AdditionalFieldsFunc
}

type VerificationFields struct {
	ID              string
	Subject         string // ${action}:${identifier} (e.g. email_verification:john.doe@example.com)
	Value           string // token/code
	ExpiresAt       string
	CreatedAt       string
	UpdatedAt       string
	SoftDeleteField string
}

type VerificationSchemaOption func(*VerificationSchema)

// NewDefaultVerificationSchema creates a new VerificationSchema with default values
func NewDefaultVerificationSchema(opts ...VerificationSchemaOption) *VerificationSchema {
	schema := &VerificationSchema{
		TableName: VerificationSchemaTableName,
		Fields: VerificationFields{
			ID:              string(SchemaIDField),
			Subject:         string(VerificationSchemaSubjectField),
			Value:           string(VerificationSchemaValueField),
			ExpiresAt:       string(VerificationSchemaExpiresAtField),
			CreatedAt:       string(VerificationSchemaCreatedAtField),
			UpdatedAt:       string(VerificationSchemaUpdatedAtField),
			SoftDeleteField: "",
		},
	}

	for _, opt := range opts {
		opt(schema)
	}

	return schema
}

func (v *VerificationSchema) GetTableName() TableName {
	return v.TableName
}

func (v *VerificationSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return v.AdditionalFields
}

func (v *VerificationSchema) GetIDField() string {
	return v.Fields.ID
}

func (v *VerificationSchema) GetSubjectField() string {
	return v.Fields.Subject
}

func (v *VerificationSchema) GetValueField() string {
	return v.Fields.Value
}

func (v *VerificationSchema) GetExpiresAtField() string {
	return v.Fields.ExpiresAt
}

func (v *VerificationSchema) GetCreatedAtField() string {
	return v.Fields.CreatedAt
}

func (v *VerificationSchema) GetUpdatedAtField() string {
	return v.Fields.UpdatedAt
}

func (v *VerificationSchema) GetSoftDeleteField() string {
	return v.Fields.SoftDeleteField
}

func (v *VerificationSchema) FromStorage(data map[string]any) *Verification {
	return &Verification{
		Subject:   data[v.GetSubjectField()].(string),
		Value:     data[v.GetValueField()].(string),
		ExpiresAt: data[v.GetExpiresAtField()].(time.Time),
		CreatedAt: data[v.GetCreatedAtField()].(time.Time),
		UpdatedAt: data[v.GetUpdatedAtField()].(time.Time),
		raw:       data,
	}
}

func (v *VerificationSchema) ToStorage(data *Verification) map[string]any {
	return map[string]any{
		v.GetSubjectField():   data.Subject,
		v.GetValueField():     data.Value,
		v.GetExpiresAtField(): data.ExpiresAt,
		v.GetCreatedAtField(): data.CreatedAt,
		v.GetUpdatedAtField(): data.UpdatedAt,
	}
}

func WithVerificationTableName(tableName TableName) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.TableName = tableName
	}
}

func WithVerificationAdditionalFields(fn AdditionalFieldsFunc) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.AdditionalFields = fn
	}
}

func WithVerificationFields(fields VerificationFields) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields = fields
	}
}

func WithVerificationFieldID(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.ID = fieldName
	}
}

func WithVerificationFieldSubject(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.Subject = fieldName
	}
}

func WithVerificationFieldValue(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.Value = fieldName
	}
}

func WithVerificationFieldExpiresAt(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.ExpiresAt = fieldName
	}
}

func WithVerificationFieldCreatedAt(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.CreatedAt = fieldName
	}
}

func WithVerificationFieldUpdatedAt(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.UpdatedAt = fieldName
	}
}

func WithVerificationFieldSoftDelete(fieldName string) VerificationSchemaOption {
	return func(s *VerificationSchema) {
		s.Fields.SoftDeleteField = fieldName
	}
}
