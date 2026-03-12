package aegis

func (s *SessionSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	return &SchemaDefinition{
		TableName: SessionSchemaTableName,
		Columns:   s.getDefaultColumns(config),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_sessions_token",
				Columns: []SchemaField{SessionSchemaTokenField},
				Unique:  true,
			},
			{
				Name:    "idx_sessions_user_id",
				Columns: []SchemaField{SessionSchemaUserIDField},
				Unique:  false,
			},
		},
		ForeignKeys: []ForeignKeyDefinition{
			{
				Name:             "fk_sessions_user_id",
				Column:           SessionSchemaUserIDField,
				ReferencedSchema: CoreSchemaUsers,
				ReferencedField:  SchemaIDField,
				OnDelete:         FKActionRestrict,
				OnUpdate:         FKActionCascade,
			},
		},
		SchemaName: CoreSchemaSessions,
		Schema:     s,
	}
}

func (s *SessionSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
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
			Name:         string(SessionSchemaTokenField),
			LogicalField: SessionSchemaTokenField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "token",
			},
		},
		{
			Name:         string(SessionSchemaUserIDField),
			LogicalField: SessionSchemaUserIDField,
			Type:         idType,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "user_id",
			},
		},
		{
			Name:         string(SessionSchemaCreatedAtField),
			LogicalField: SessionSchemaCreatedAtField,
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			DefaultValue: string(DatabaseDefaultValueNow),
			Tags: map[string]string{
				"json": "created_at",
			},
		},
		{
			Name:         string(SessionSchemaExpiresAtField),
			LogicalField: SessionSchemaExpiresAtField,
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "expires_at",
			},
		},
		{
			Name:         string(SessionSchemaLastAccessField),
			LogicalField: SessionSchemaLastAccessField,
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "last_access",
			},
		},
		{
			Name:         string(SessionSchemaMetadataField),
			LogicalField: SessionSchemaMetadataField,
			Type:         ColumnTypeMapStringAny,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "metadata",
			},
		},
	}
}
