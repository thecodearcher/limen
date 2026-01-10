package aegis

import (
	"time"
)

type User struct {
	ID              any        `json:"id"`
	Email           string     `json:"email"`
	Password        string     `json:"-"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	raw             map[string]any
}

// Raw returns the user raw data as returned from the database
func (u User) Raw() map[string]any {
	return u.raw
}

func (c User) TableName() string {
	return string(UserSchemaTableName)
}

type UserSchema struct {
	BaseSchema
	// If true, the schema will include the first name and last name fields
	includeNameFields bool

	// If true, the schema will include the created at and updated at fields
	includeTimestampFields bool

	// A function to serialize the model to a json object for returning to the client
	Serializer func(data *User) map[string]any
}

type SchemaConfigUserOption func(*SchemaConfig, *UserSchema)

func newDefaultUserSchema(c *SchemaConfig, opts ...SchemaConfigUserOption) *UserSchema {
	schema := &UserSchema{
		BaseSchema:             BaseSchema{},
		includeNameFields:      true,
		includeTimestampFields: true,
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (u *UserSchema) GetEmailField() string {
	return u.GetField(string(UserSchemaEmailField))
}

func (u *UserSchema) GetPasswordField() string {
	return u.GetField(string(UserSchemaPasswordField))
}

func (u *UserSchema) GetEmailVerifiedAtField() string {
	return u.GetField(string(UserSchemaEmailVerifiedAtField))
}

func (u *UserSchema) FromStorage(data map[string]any) Model {
	return &User{
		ID:              data[u.GetIDField()],
		Email:           data[u.GetEmailField()].(string),
		Password:        data[u.GetPasswordField()].(string),
		EmailVerifiedAt: getNullableValue[time.Time](data[u.GetEmailVerifiedAtField()]),
		raw:             data,
	}
}

func (u *UserSchema) ToStorage(data Model) map[string]any {
	user := data.(*User)
	return map[string]any{
		u.GetEmailField():           user.Email,
		u.GetPasswordField():        user.Password,
		u.GetEmailVerifiedAtField(): user.EmailVerifiedAt,
	}
}

func (u *UserSchema) Serialize(data *User) map[string]any {
	if u.Serializer != nil {
		return u.Serializer(data)
	}
	raw := data.Raw()
	delete(raw, u.GetPasswordField())
	return raw
}

func WithUserTableName(tableName SchemaTableName) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaTableName(CoreSchemaUsers, tableName)
	}
}

func WithUserAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigUserOption {
	return func(c *SchemaConfig, u *UserSchema) {
		u.additionalFields = fn
	}
}

func WithUserSerializer(serializer func(data *User) map[string]any) SchemaConfigUserOption {
	return func(c *SchemaConfig, u *UserSchema) {
		u.Serializer = serializer
	}
}

func WithUserFieldID(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(SchemaIDField), fieldName)
	}
}

func WithUserFieldEmail(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(UserSchemaEmailField), fieldName)
	}
}

func WithUserFieldPassword(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(UserSchemaPasswordField), fieldName)
	}
}

func WithUserFieldEmailVerifiedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(UserSchemaEmailVerifiedAtField), fieldName)
	}
}

func WithUserFieldSoftDelete(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(SchemaSoftDeleteField), fieldName)
	}
}

func WithUserIncludeNameFields(include bool) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		u.includeNameFields = include
	}
}

func WithUserFirstNameField(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(UserSchemaFirstNameField), fieldName)
	}
}

func WithUserLastNameField(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(UserSchemaLastNameField), fieldName)
	}
}

func WithUserFieldCreatedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(SchemaCreatedAtField), fieldName)
	}
}

func WithUserFieldUpdatedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, string(SchemaUpdatedAtField), fieldName)
	}
}

func (u *UserSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	tableName := UserSchemaTableName
	return &SchemaDefinition{
		TableName: &tableName,
		Columns:   u.getDefaultColumns(config),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_users_email",
				Columns: []string{string(UserSchemaEmailField)},
				Unique:  true,
			},
		},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  string(CoreSchemaUsers),
		Extends:     nil,
		Schema:      u,
	}
}

func (u *UserSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
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
			Name:         string(UserSchemaEmailField),
			LogicalField: string(UserSchemaEmailField),
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "email",
			},
		},
		{
			Name:         string(UserSchemaPasswordField),
			LogicalField: string(UserSchemaPasswordField),
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "-",
			},
		},
		{
			Name:         string(UserSchemaEmailVerifiedAtField),
			LogicalField: string(UserSchemaEmailVerifiedAtField),
			Type:         ColumnTypeTime,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "email_verified_at",
			},
		},
	}

	if u.includeNameFields {
		fields = append(fields,
			ColumnDefinition{
				Name:         string(UserSchemaFirstNameField),
				LogicalField: string(UserSchemaFirstNameField),
				Type:         ColumnTypeString,
				IsNullable:   false,
				IsPrimaryKey: false,
				Tags: map[string]string{
					"json": "first_name",
				},
			},
			ColumnDefinition{
				Name:         string(UserSchemaLastNameField),
				LogicalField: string(UserSchemaLastNameField),
				Type:         ColumnTypeString,
				IsNullable:   true,
				IsPrimaryKey: false,
				Tags: map[string]string{
					"json": "last_name",
				},
			},
		)
	}

	if u.includeTimestampFields {
		fields = addTimestampFields(fields)
	}

	softDeleteField := config.getCoreSchemaCustomizationField(CoreSchemaUsers, string(SchemaSoftDeleteField))
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
