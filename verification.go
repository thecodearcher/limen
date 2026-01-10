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
	return v.GetField(string(VerificationSchemaSubjectField))
}

func (v *VerificationSchema) GetValueField() string {
	return v.GetField(string(VerificationSchemaValueField))
}

func (v *VerificationSchema) GetExpiresAtField() string {
	return v.GetField(string(VerificationSchemaExpiresAtField))
}

func (v *VerificationSchema) GetCreatedAtField() string {
	return v.GetField(string(SchemaCreatedAtField))
}

func (v *VerificationSchema) GetUpdatedAtField() string {
	return v.GetField(string(SchemaUpdatedAtField))
}

func (v *VerificationSchema) FromStorage(data map[string]any) Model {
	return &Verification{
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
		s.setCoreSchemaField(CoreSchemaVerifications, string(SchemaIDField), fieldName)
	}
}

func WithVerificationFieldSubject(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, string(VerificationSchemaSubjectField), fieldName)
	}
}

func WithVerificationFieldValue(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, string(VerificationSchemaValueField), fieldName)
	}
}

func WithVerificationFieldExpiresAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, string(VerificationSchemaExpiresAtField), fieldName)
	}
}

func WithVerificationFieldCreatedAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, string(SchemaCreatedAtField), fieldName)
	}
}

func WithVerificationFieldUpdatedAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, string(SchemaUpdatedAtField), fieldName)
	}
}

func WithVerificationFieldSoftDelete(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, string(SchemaSoftDeleteField), fieldName)
	}
}

func (v *VerificationSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	fields := v.getDefaultColumns(config)
	tableName := VerificationSchemaTableName

	return &SchemaDefinition{
		TableName: &tableName,
		Columns:   fields,
		Indexes: []IndexDefinition{
			{
				Name:    "idx_verifications_value",
				Columns: []string{string(VerificationSchemaValueField)},
				Unique:  true,
			},
			{
				Name:    "idx_verifications_subject",
				Columns: []string{string(VerificationSchemaSubjectField)},
				Unique:  false,
			},
		},
		SchemaName: string(CoreSchemaVerifications),
		Extends:    nil,
		Schema:     v,
	}
}

func (v *VerificationSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
	idType := config.GetIDColumnType()

	fields := []ColumnDefinition{
		{
			Name:         string(SchemaIDField),
			LogicalField: string(SchemaIDField),
			Type:         idType,
			IsNullable:   false,
			IsPrimaryKey: true,
			Tags: map[string]string{
				"json": "id",
			},
		},
		{
			Name:         string(VerificationSchemaSubjectField),
			LogicalField: string(VerificationSchemaSubjectField),
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "subject",
			},
		},
		{
			Name:         string(VerificationSchemaValueField),
			LogicalField: string(VerificationSchemaValueField),
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "value",
			},
		},
		{
			Name:         string(VerificationSchemaExpiresAtField),
			LogicalField: string(VerificationSchemaExpiresAtField),
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "expires_at",
			},
		},
	}

	fields = addTimestampFields(fields)

	softDeleteField := config.getCoreSchemaCustomizationField(CoreSchemaVerifications, string(SchemaSoftDeleteField))
	if softDeleteField != "" {
		fields = append(fields, ColumnDefinition{
			Name:         softDeleteField,
			LogicalField: string(SchemaSoftDeleteField),
			Type:         ColumnTypeTime,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": softDeleteField,
			},
		})
	}

	return fields
}
