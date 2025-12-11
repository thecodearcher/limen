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
	BaseSchema
}

type SchemaConfigRateLimitOption func(*SchemaConfig, *RateLimitSchema)

func newDefaultRateLimitSchema(c *SchemaConfig, opts ...SchemaConfigRateLimitOption) *RateLimitSchema {
	schema := &RateLimitSchema{
		BaseSchema: BaseSchema{},
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (r *RateLimitSchema) GetIDField() string {
	return r.GetField(string(SchemaIDField))
}

func (r *RateLimitSchema) GetKeyField() string {
	return r.GetField(string(RateLimitSchemaKeyField))
}

func (r *RateLimitSchema) GetCountField() string {
	return r.GetField(string(RateLimitSchemaCountField))
}

func (r *RateLimitSchema) GetLastRequestAtField() string {
	return r.GetField(string(RateLimitSchemaLastRequestAtField))
}

func (r *RateLimitSchema) FromStorage(data map[string]any) Model {
	return &RateLimit{
		ID:            data[r.GetIDField()],
		Key:           data[r.GetKeyField()].(string),
		Count:         int(data[r.GetCountField()].(int32)),
		LastRequestAt: data[r.GetLastRequestAtField()].(int64),
		raw:           data,
	}
}

func (r *RateLimitSchema) ToStorage(data Model) map[string]any {
	rateLimit := data.(*RateLimit)
	return map[string]any{
		r.GetKeyField():           rateLimit.Key,
		r.GetCountField():         rateLimit.Count,
		r.GetLastRequestAtField(): rateLimit.LastRequestAt,
	}
}

func WithRateLimitTableName(tableName SchemaTableName) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaTableName(CoreSchemaRateLimits, tableName)
	}
}

func WithRateLimitFieldID(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, string(SchemaIDField), fieldName)
	}
}

func WithRateLimitFieldKey(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, string(RateLimitSchemaKeyField), fieldName)
	}
}

func WithRateLimitFieldCount(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, string(RateLimitSchemaCountField), fieldName)
	}
}

func WithRateLimitFieldLastRequestAt(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, string(RateLimitSchemaLastRequestAtField), fieldName)
	}
}

func (r *RateLimitSchema) Introspect() SchemaIntrospector {
	tableName := RateLimitSchemaTableName
	return &SchemaDefinition{
		TableName: &tableName,
		Columns:   r.getDefaultColumns(),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_rate_limits_key",
				Columns: []string{r.GetKeyField()},
				Unique:  true,
			},
		},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  string(CoreSchemaRateLimits),
		Extends:     nil,
		Schema:      r,
	}
}

func (r *RateLimitSchema) getDefaultColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{
			Name:         string(SchemaIDField),
			LogicalField: string(SchemaIDField),
			Type:         ColumnTypeAny,
			IsNullable:   false,
			IsPrimaryKey: true,
			Tags: map[string]string{
				"json": "id",
			},
		},
		{
			Name:         string(RateLimitSchemaKeyField),
			LogicalField: string(RateLimitSchemaKeyField),
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "key",
			},
		},
		{
			Name:         string(RateLimitSchemaCountField),
			LogicalField: string(RateLimitSchemaCountField),
			Type:         ColumnTypeInt,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "count",
			},
		},
		{
			Name:         string(RateLimitSchemaLastRequestAtField),
			LogicalField: string(RateLimitSchemaLastRequestAtField),
			Type:         ColumnTypeInt64,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "last_request_at",
			},
		},
	}
}
