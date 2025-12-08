package aegis

import "time"

type RateLimit struct {
	ID            any
	Key           string
	Count         int
	LastRequestAt int64
	raw           map[string]any
}

func (r *RateLimit) Touch() {
	r.LastRequestAt = time.Now().UnixMilli()
	r.Count = r.Count + 1
}

func (r *RateLimit) ResetCounter() {
	r.Count = 0
}

func (r RateLimit) Raw() map[string]any {
	return r.raw
}

type RateLimitSchema struct {
	// name of the table in the database
	TableName TableName

	// mapping of the rate limit schema to the database columns
	Fields RateLimitFields
}

type RateLimitFields struct {
	ID            string
	Key           string
	Count         string
	LastRequestAt string
}

type RateLimitSchemaOption func(*RateLimitSchema)

// NewDefaultRateLimitSchema creates a new RateLimitSchema with default values
func NewDefaultRateLimitSchema(opts ...RateLimitSchemaOption) *RateLimitSchema {
	schema := &RateLimitSchema{
		TableName: RateLimitSchemaTableName,
		Fields: RateLimitFields{
			ID:            string(SchemaIDField),
			Key:           string(RateLimitSchemaKeyField),
			Count:         string(RateLimitSchemaCountField),
			LastRequestAt: string(RateLimitSchemaLastRequestAtField),
		},
	}

	for _, opt := range opts {
		opt(schema)
	}

	return schema
}

func (r *RateLimitSchema) GetTableName() TableName {
	return r.TableName
}

func (r *RateLimitSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return nil
}

func (r *RateLimitSchema) GetSoftDeleteField() string {
	return ""
}

func (r *RateLimitSchema) GetIDField() string {
	return r.Fields.ID
}

func (r *RateLimitSchema) GetKeyField() string {
	return r.Fields.Key
}

func (r *RateLimitSchema) GetCountField() string {
	return r.Fields.Count
}

func (r *RateLimitSchema) GetLastRequestAtField() string {
	return r.Fields.LastRequestAt
}

func (r *RateLimitSchema) FromStorage(data map[string]any) *RateLimit {
	return &RateLimit{
		ID:            data[r.GetIDField()],
		Key:           data[r.GetKeyField()].(string),
		Count:         int(data[r.GetCountField()].(int32)),
		LastRequestAt: data[r.GetLastRequestAtField()].(int64),
		raw:           data,
	}
}

func (r *RateLimitSchema) ToStorage(data *RateLimit) map[string]any {
	return map[string]any{
		r.GetKeyField():           data.Key,
		r.GetCountField():         data.Count,
		r.GetLastRequestAtField(): data.LastRequestAt,
	}
}

func WithRateLimitTableName(tableName TableName) RateLimitSchemaOption {
	return func(s *RateLimitSchema) {
		s.TableName = tableName
	}
}

func WithRateLimitFields(fields RateLimitFields) RateLimitSchemaOption {
	return func(s *RateLimitSchema) {
		s.Fields = fields
	}
}

func WithRateLimitFieldID(fieldName string) RateLimitSchemaOption {
	return func(s *RateLimitSchema) {
		s.Fields.ID = fieldName
	}
}

func WithRateLimitFieldKey(fieldName string) RateLimitSchemaOption {
	return func(s *RateLimitSchema) {
		s.Fields.Key = fieldName
	}
}

func WithRateLimitFieldCount(fieldName string) RateLimitSchemaOption {
	return func(s *RateLimitSchema) {
		s.Fields.Count = fieldName
	}
}

func WithRateLimitFieldLastRequestAt(fieldName string) RateLimitSchemaOption {
	return func(s *RateLimitSchema) {
		s.Fields.LastRequestAt = fieldName
	}
}

// Introspect implements SchemaIntrospector for RateLimitSchema
func (r *RateLimitSchema) Introspect() SchemaIntrospector {
	return &rateLimitSchemaIntrospector{schema: r}
}

type rateLimitSchemaIntrospector struct {
	schema *RateLimitSchema
}

func (r *rateLimitSchemaIntrospector) GetTableName() TableName {
	return r.schema.TableName
}

func (r *rateLimitSchemaIntrospector) GetFields() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:         r.schema.Fields.ID,
			Type:         "any",
			IsNullable:   false,
			IsPrimaryKey: true,
			Tags: map[string]string{
				"json": "id",
			},
		},
		{
			Name:         r.schema.Fields.Key,
			Type:         "string",
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "key",
			},
		},
		{
			Name:         r.schema.Fields.Count,
			Type:         "int",
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "count",
			},
		},
		{
			Name:         r.schema.Fields.LastRequestAt,
			Type:         "int64",
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "last_request_at",
			},
		},
	}
}

func (r *rateLimitSchemaIntrospector) GetIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:    "idx_rate_limits_key",
			Columns: []string{r.schema.Fields.Key},
			Unique:  true,
		},
	}
}

func (r *rateLimitSchemaIntrospector) GetForeignKeys() []ForeignKeyDefinition {
	return []ForeignKeyDefinition{}
}

func (r *rateLimitSchemaIntrospector) GetExtends() string {
	return ""
}
