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
	return u.GetField(UserSchemaEmailField)
}

func (u *UserSchema) GetPasswordField() string {
	return u.GetField(UserSchemaPasswordField)
}

func (u *UserSchema) GetEmailVerifiedAtField() string {
	return u.GetField(UserSchemaEmailVerifiedAtField)
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

func (u *UserSchema) Serialize(data Model) map[string]any {
	if u.BaseSchema.Serializer != nil {
		return u.BaseSchema.Serializer(data)
	}
	raw := data.Raw()
	delete(raw, u.GetIDField())
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

// WithUserSerializer overrides the default user response serializer.
// When not provided, Aegis serializes from user.Raw() and removes the password field.
func WithUserSerializer(serializer func(data *User) map[string]any) SchemaConfigUserOption {
	return func(c *SchemaConfig, u *UserSchema) {
		u.BaseSchema.Serializer = func(data Model) map[string]any {
			return serializer(data.(*User))
		}
	}
}

func WithUserFieldID(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaIDField, fieldName)
	}
}

func WithUserFieldEmail(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaEmailField, fieldName)
	}
}

func WithUserFieldPassword(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaPasswordField, fieldName)
	}
}

func WithUserFieldEmailVerifiedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaEmailVerifiedAtField, fieldName)
	}
}

func WithUserFieldSoftDelete(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaSoftDeleteField, fieldName)
	}
}

func WithUserIncludeNameFields(include bool) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		u.includeNameFields = include
	}
}

func WithUserFirstNameField(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaFirstNameField, fieldName)
	}
}

func WithUserLastNameField(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaLastNameField, fieldName)
	}
}

func WithUserFieldCreatedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaCreatedAtField, fieldName)
	}
}

func WithUserFieldUpdatedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaUpdatedAtField, fieldName)
	}
}

func (u *UserSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	return &SchemaDefinition{
		TableName: UserSchemaTableName,
		Columns:   u.getDefaultColumns(config),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_users_email",
				Columns: []SchemaField{UserSchemaEmailField},
				Unique:  true,
			},
		},
		ForeignKeys: []ForeignKeyDefinition{},
		SchemaName:  CoreSchemaUsers,
		Schema:      u,
	}
}

func (u *UserSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
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
			Name:         string(UserSchemaEmailField),
			LogicalField: UserSchemaEmailField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "email",
			},
		},
		{
			Name:         string(UserSchemaPasswordField),
			LogicalField: UserSchemaPasswordField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "-",
			},
		},
		{
			Name:         string(UserSchemaEmailVerifiedAtField),
			LogicalField: UserSchemaEmailVerifiedAtField,
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
				LogicalField: UserSchemaFirstNameField,
				Type:         ColumnTypeString,
				IsNullable:   false,
				IsPrimaryKey: false,
				Tags: map[string]string{
					"json": "first_name",
				},
			},
			ColumnDefinition{
				Name:         string(UserSchemaLastNameField),
				LogicalField: UserSchemaLastNameField,
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

	fields = addSoftDeleteField(fields, config, CoreSchemaUsers)

	return fields
}
