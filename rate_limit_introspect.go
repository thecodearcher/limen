package limen

func (r *RateLimitSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	tableName := RateLimitSchemaTableName
	return &SchemaDefinition{
		TableName: tableName,
		Columns:   r.getDefaultColumns(config),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_rate_limits_key",
				Columns: []SchemaField{RateLimitSchemaKeyField},
				Unique:  true,
			},
		},
		SchemaName: CoreSchemaRateLimits,
		Schema:     r,
	}
}

func (r *RateLimitSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
	idType := config.GetIDColumnType()

	return []ColumnDefinition{
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
			Name:         string(RateLimitSchemaKeyField),
			LogicalField: RateLimitSchemaKeyField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "key",
			},
		},
		{
			Name:         string(RateLimitSchemaCountField),
			LogicalField: RateLimitSchemaCountField,
			Type:         ColumnTypeInt,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "count",
			},
		},
		{
			Name:         string(RateLimitSchemaLastRequestAtField),
			LogicalField: RateLimitSchemaLastRequestAtField,
			Type:         ColumnTypeInt64,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "last_request_at",
			},
		},
	}
}
