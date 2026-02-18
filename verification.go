package aegis

import "time"

type Verification struct {
	ID        any
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
	BaseSchema
}

type SchemaConfigVerificationOption func(*SchemaConfig, *VerificationSchema)

func newDefaultVerificationSchema(c *SchemaConfig, opts ...SchemaConfigVerificationOption) *VerificationSchema {
	schema := &VerificationSchema{
		BaseSchema: BaseSchema{},
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (v *VerificationSchema) GetSubjectField() string {
	return v.GetField(VerificationSchemaSubjectField)
}

func (v *VerificationSchema) GetValueField() string {
	return v.GetField(VerificationSchemaValueField)
}

func (v *VerificationSchema) GetExpiresAtField() string {
	return v.GetField(VerificationSchemaExpiresAtField)
}

func (v *VerificationSchema) GetCreatedAtField() string {
	return v.GetField(SchemaCreatedAtField)
}

func (v *VerificationSchema) GetUpdatedAtField() string {
	return v.GetField(SchemaUpdatedAtField)
}

func (v *VerificationSchema) FromStorage(data map[string]any) Model {
	return &Verification{
		ID:        data[v.GetIDField()],
		Subject:   data[v.GetSubjectField()].(string),
		Value:     data[v.GetValueField()].(string),
		ExpiresAt: data[v.GetExpiresAtField()].(time.Time),
		CreatedAt: data[v.GetCreatedAtField()].(time.Time),
		UpdatedAt: data[v.GetUpdatedAtField()].(time.Time),
		raw:       data,
	}
}

func (v *VerificationSchema) ToStorage(data Model) map[string]any {
	verification := data.(*Verification)
	return map[string]any{
		v.GetSubjectField():   verification.Subject,
		v.GetValueField():     verification.Value,
		v.GetExpiresAtField(): verification.ExpiresAt,
		v.GetCreatedAtField(): verification.CreatedAt,
		v.GetUpdatedAtField(): verification.UpdatedAt,
	}
}

func WithVerificationTableName(tableName SchemaTableName) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaTableName(CoreSchemaVerifications, tableName)
	}
}

func WithVerificationAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigVerificationOption {
	return func(c *SchemaConfig, v *VerificationSchema) {
		v.additionalFields = fn
	}
}

func WithVerificationFieldID(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaIDField, fieldName)
	}
}

func WithVerificationFieldSubject(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, VerificationSchemaSubjectField, fieldName)
	}
}

func WithVerificationFieldValue(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, VerificationSchemaValueField, fieldName)
	}
}

func WithVerificationFieldExpiresAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, VerificationSchemaExpiresAtField, fieldName)
	}
}

func WithVerificationFieldCreatedAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaCreatedAtField, fieldName)
	}
}

func WithVerificationFieldUpdatedAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaUpdatedAtField, fieldName)
	}
}

func WithVerificationFieldSoftDelete(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaSoftDeleteField, fieldName)
	}
}

func (v *VerificationSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	fields := v.getDefaultColumns(config)

	return &SchemaDefinition{
		TableName: VerificationSchemaTableName,
		Columns:   fields,
		Indexes: []IndexDefinition{
			{
				Name:    "idx_verifications_value",
				Columns: []SchemaField{VerificationSchemaValueField},
				Unique:  true,
			},
			{
				Name:    "idx_verifications_subject",
				Columns: []SchemaField{VerificationSchemaSubjectField},
				Unique:  false,
			},
		},
		SchemaName: CoreSchemaVerifications,
		Schema:     v,
	}
}

func (v *VerificationSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
	idType := config.GetIDColumnType()

	fields := []ColumnDefinition{
		{
			Name:         string(SchemaIDField),
			LogicalField: SchemaIDField,
			Type:         idType,
			IsNullable:   false,
			IsPrimaryKey: true,
			Tags: map[string]string{
				"json": "id",
			},
		},
		{
			Name:         string(VerificationSchemaSubjectField),
			LogicalField: VerificationSchemaSubjectField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "subject",
			},
		},
		{
			Name:         string(VerificationSchemaValueField),
			LogicalField: VerificationSchemaValueField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "value",
			},
		},
		{
			Name:         string(VerificationSchemaExpiresAtField),
			LogicalField: VerificationSchemaExpiresAtField,
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "expires_at",
			},
		},
	}

	fields = addTimestampFields(fields)

	fields = addSoftDeleteField(fields, config, CoreSchemaVerifications)

	return fields
}
