package aegis

import (
	"strings"
	"time"
)

type SchemaConfigAccountOption func(*SchemaConfig, *AccountSchema)

type AccountSchema struct {
	BaseSchema
}

func (s *AccountSchema) Serialize(data Model) map[string]any {
	if s.BaseSchema.Serializer != nil {
		return s.BaseSchema.Serializer(data)
	}
	raw := data.Raw()
	delete(raw, s.GetIDField())
	delete(raw, s.GetAccessTokenField())
	delete(raw, s.GetRefreshTokenField())
	delete(raw, s.GetIDTokenField())
	delete(raw, s.GetUserIDField())

	scope := raw[s.GetScopeField()].(string)
	scope = strings.Join(strings.Split(scope, " "), ",")
	scopes := strings.Split(scope, ",")
	delete(raw, s.GetScopeField())
	raw["scopes"] = scopes
	return raw
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
