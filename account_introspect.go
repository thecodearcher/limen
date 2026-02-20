package aegis

func (s *AccountSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	return &SchemaDefinition{
		TableName: AccountSchemaTableName,
		Columns:   s.getDefaultColumns(config),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_accounts_user_id_provider",
				Columns: []SchemaField{AccountSchemaUserIDField},
			},
			{
				Name:    "idx_accounts_provider_provider_account_id",
				Columns: []SchemaField{AccountSchemaProviderField, AccountSchemaProviderAccountIDField},
				Unique:  true,
			},
		},
		ForeignKeys: []ForeignKeyDefinition{
			{
				Name:             "fk_accounts_users_user_id",
				Column:           AccountSchemaUserIDField,
				ReferencedSchema: CoreSchemaUsers,
				ReferencedField:  SchemaIDField,
				OnDelete:         FKActionCascade,
				OnUpdate:         FKActionCascade,
			},
		},
		SchemaName: CoreSchemaAccounts,
		Schema:     s,
	}
}

func (s *AccountSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
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
			Name:         string(AccountSchemaUserIDField),
			LogicalField: AccountSchemaUserIDField,
			Type:         idType,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "user_id",
			},
		},
		{
			Name:         string(AccountSchemaProviderField),
			LogicalField: AccountSchemaProviderField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "provider",
			},
		},
		{
			Name:         string(AccountSchemaProviderAccountIDField),
			LogicalField: AccountSchemaProviderAccountIDField,
			Type:         ColumnTypeString,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "provider_account_id",
			},
		},
		{
			Name:         string(AccountSchemaAccessTokenField),
			LogicalField: AccountSchemaAccessTokenField,
			Type:         ColumnTypeText,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "access_token",
			},
		},
		{
			Name:         string(AccountSchemaRefreshTokenField),
			LogicalField: AccountSchemaRefreshTokenField,
			Type:         ColumnTypeText,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "refresh_token",
			},
		},
		{
			Name:         string(AccountSchemaAccessTokenExpiresAtField),
			LogicalField: AccountSchemaAccessTokenExpiresAtField,
			Type:         ColumnTypeTime,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "access_token_expires_at",
			},
		},
		{
			Name:         string(AccountSchemaScopeField),
			LogicalField: AccountSchemaScopeField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "scope",
			},
		},
		{
			Name:         string(AccountSchemaIDTokenField),
			LogicalField: AccountSchemaIDTokenField,
			Type:         ColumnTypeText,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "id_token",
			},
		},
	}

	fields = addTimestampFields(fields)
	fields = addSoftDeleteField(fields, config, CoreSchemaAccounts)

	return fields
}
