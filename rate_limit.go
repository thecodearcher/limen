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
	TableName SchemaTableName

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

func (r *RateLimitSchema) GetTableName() SchemaTableName {
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

func WithRateLimitTableName(tableName SchemaTableName) RateLimitSchemaOption {
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
	return NewIntrospector(
		r,
		r.TableName,
		string(CoreSchemaRateLimits),
		func(schema *RateLimitSchema) []ColumnDefinition {
			return []ColumnDefinition{
				{
					Name:         schema.Fields.ID,
					LogicalField: string(SchemaIDField),
					Type:         ColumnTypeAny,
					IsNullable:   false,
					IsPrimaryKey: true,
					Tags: map[string]string{
						"json": "id",
					},
				},
				{
					Name:         schema.Fields.Key,
					LogicalField: string(RateLimitSchemaKeyField),
					Type:         ColumnTypeString,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "key",
					},
				},
				{
					Name:         schema.Fields.Count,
					LogicalField: string(RateLimitSchemaCountField),
					Type:         ColumnTypeInt,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "count",
					},
				},
				{
					Name:         schema.Fields.LastRequestAt,
					LogicalField: string(RateLimitSchemaLastRequestAtField),
					Type:         ColumnTypeInt64,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "last_request_at",
					},
				},
			}
		},
		func(schema *RateLimitSchema) []IndexDefinition {
			return []IndexDefinition{
				{
					Name:    "idx_rate_limits_key",
					Columns: []string{schema.Fields.Key},
					Unique:  true,
				},
			}
		},
		func(schema *RateLimitSchema) []ForeignKeyDefinition {
			return []ForeignKeyDefinition{}
		},
		func(schema *RateLimitSchema) *CoreSchemaName {
			return nil
		},
	)
}
