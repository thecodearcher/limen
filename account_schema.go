package aegis

import (
	"time"
)

type SchemaConfigAccountOption func(*SchemaConfig, *AccountSchema)

type AccountSchema struct {
	BaseSchema
}

func WithAccountTableName(tableName SchemaTableName) SchemaConfigAccountOption {
	return func(c *SchemaConfig, s *AccountSchema) {
		c.setCoreSchemaTableName(CoreSchemaAccounts, tableName)
	}
}

func newDefaultAccountSchema(c *SchemaConfig, opts ...SchemaConfigAccountOption) *AccountSchema {
	schema := &AccountSchema{
		BaseSchema: BaseSchema{},
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (s *AccountSchema) GetUserIDField() string {
	return s.GetField(AccountSchemaUserIDField)
}

func (s *AccountSchema) GetProviderField() string {
	return s.GetField(AccountSchemaProviderField)
}

func (s *AccountSchema) GetProviderAccountIDField() string {
	return s.GetField(AccountSchemaProviderAccountIDField)
}

func (s *AccountSchema) GetAccessTokenField() string {
	return s.GetField(AccountSchemaAccessTokenField)
}

func (s *AccountSchema) GetRefreshTokenField() string {
	return s.GetField(AccountSchemaRefreshTokenField)
}

func (s *AccountSchema) GetAccessTokenExpiresAtField() string {
	return s.GetField(AccountSchemaAccessTokenExpiresAtField)
}

func (s *AccountSchema) GetScopeField() string {
	return s.GetField(AccountSchemaScopeField)
}

func (s *AccountSchema) GetIDTokenField() string {
	return s.GetField(AccountSchemaIDTokenField)
}

func (s *AccountSchema) GetCreatedAtField() string {
	return s.GetField(SchemaCreatedAtField)
}

func (s *AccountSchema) GetUpdatedAtField() string {
	return s.GetField(SchemaUpdatedAtField)
}

func (s *AccountSchema) ToStorage(data Model) map[string]any {
	acc := data.(*Account)
	m := map[string]any{
		s.GetUserIDField():            acc.UserID,
		s.GetProviderField():          acc.Provider,
		s.GetProviderAccountIDField(): acc.ProviderAccountID,
		s.GetAccessTokenField():       acc.AccessToken,
		s.GetScopeField():             acc.Scope,
		s.GetCreatedAtField():         acc.CreatedAt,
		s.GetUpdatedAtField():         acc.UpdatedAt,
	}
	if acc.RefreshToken != "" {
		m[s.GetRefreshTokenField()] = acc.RefreshToken
	}
	if acc.AccessTokenExpiresAt != nil {
		m[s.GetAccessTokenExpiresAtField()] = *acc.AccessTokenExpiresAt
	}
	if acc.IDToken != "" {
		m[s.GetIDTokenField()] = acc.IDToken
	}
	return m
}

func (s *AccountSchema) FromStorage(data map[string]any) Model {
	acc := &Account{
		ID:                   data[s.GetIDField()],
		UserID:               data[s.GetUserIDField()],
		Provider:             getString(data[s.GetProviderField()]),
		ProviderAccountID:    getString(data[s.GetProviderAccountIDField()]),
		AccessToken:          getString(data[s.GetAccessTokenField()]),
		RefreshToken:         getString(data[s.GetRefreshTokenField()]),
		AccessTokenExpiresAt: getNullableValue[time.Time](data[s.GetAccessTokenExpiresAtField()]),
		IDToken:              getString(data[s.GetIDTokenField()]),
		CreatedAt:            getTime(data[s.GetCreatedAtField()]),
		UpdatedAt:            getTime(data[s.GetUpdatedAtField()]),
		Scope:                getString(data[s.GetScopeField()]),
		raw:                  data,
	}
	return acc
}

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
