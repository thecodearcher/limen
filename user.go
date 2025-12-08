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
	// name of the table in the database
	TableName TableName
	// A function to return a map of additional fields to be added to the schema when creating a record. e.g:
	//  func(ctx context.Context) map[string]any {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }
	//	 }
	// NOTE: fields here will override the global additional fields function.
	AdditionalFields AdditionalFieldsFunc
	// mapping of the user schema to the database columns
	Fields UserFields

	// A function to serialize the model to a json object for returning to the client
	Serializer func(data *User) map[string]any
}

type UserFields struct {
	ID              string
	Email           string
	Password        string
	EmailVerifiedAt string
	SoftDeleteField string
}

type UserSchemaOption func(*UserSchema)

// NewDefaultUserSchema creates a new UserSchema with default values
func NewDefaultUserSchema(opts ...UserSchemaOption) *UserSchema {
	schema := &UserSchema{
		TableName: UserSchemaTableName,
		Fields: UserFields{
			ID:              string(SchemaIDField),
			Email:           string(UserSchemaEmailField),
			Password:        string(UserSchemaPasswordField),
			EmailVerifiedAt: string(UserSchemaEmailVerifiedAtField),
			SoftDeleteField: string(UserSchemaSoftDeleteField),
		},
	}

	for _, opt := range opts {
		opt(schema)
	}

	return schema
}

func (u *UserSchema) GetTableName() TableName {
	return u.TableName
}

func (u *UserSchema) GetSoftDeleteField() string {
	return u.Fields.SoftDeleteField
}

func (u *UserSchema) GetIDField() string {
	return u.Fields.ID
}

func (u *UserSchema) GetEmailField() string {
	return u.Fields.Email
}

func (u *UserSchema) GetPasswordField() string {
	return u.Fields.Password
}

func (u *UserSchema) GetEmailVerifiedAtField() string {
	return u.Fields.EmailVerifiedAt
}

func (u *UserSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return u.AdditionalFields
}

func (u *UserSchema) FromStorage(data map[string]any) *User {
	return &User{
		ID:              data[u.GetIDField()],
		Email:           data[u.GetEmailField()].(string),
		Password:        data[u.GetPasswordField()].(string),
		EmailVerifiedAt: getNullableValue[time.Time](data[u.GetEmailVerifiedAtField()]),
		raw:             data,
	}
}

func (u *UserSchema) ToStorage(data *User) map[string]any {
	return map[string]any{
		u.GetEmailField():           data.Email,
		u.GetPasswordField():        data.Password,
		u.GetEmailVerifiedAtField(): data.EmailVerifiedAt,
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

func WithUserTableName(tableName TableName) UserSchemaOption {
	return func(s *UserSchema) {
		s.TableName = tableName
	}
}

func WithUserAdditionalFields(fn AdditionalFieldsFunc) UserSchemaOption {
	return func(s *UserSchema) {
		s.AdditionalFields = fn
	}
}

func WithUserSerializer(serializer func(data *User) map[string]any) UserSchemaOption {
	return func(s *UserSchema) {
		s.Serializer = serializer
	}
}

func WithUserFields(fields UserFields) UserSchemaOption {
	return func(s *UserSchema) {
		s.Fields = fields
	}
}

func WithUserFieldID(fieldName string) UserSchemaOption {
	return func(s *UserSchema) {
		s.Fields.ID = fieldName
	}
}

func WithUserFieldEmail(fieldName string) UserSchemaOption {
	return func(s *UserSchema) {
		s.Fields.Email = fieldName
	}
}

func WithUserFieldPassword(fieldName string) UserSchemaOption {
	return func(s *UserSchema) {
		s.Fields.Password = fieldName
	}
}

func WithUserFieldEmailVerifiedAt(fieldName string) UserSchemaOption {
	return func(s *UserSchema) {
		s.Fields.EmailVerifiedAt = fieldName
	}
}

func WithUserFieldSoftDelete(fieldName string) UserSchemaOption {
	return func(s *UserSchema) {
		s.Fields.SoftDeleteField = fieldName
	}
}

// Introspect implements SchemaIntrospector for UserSchema
func (u *UserSchema) Introspect() SchemaIntrospector {
	return &userSchemaIntrospector{schema: u}
}

type userSchemaIntrospector struct {
	schema *UserSchema
}

func (u *userSchemaIntrospector) GetTableName() TableName {
	return u.schema.TableName
}

func (u *userSchemaIntrospector) GetFields() []FieldDefinition {
	fields := []FieldDefinition{
		{
			Name:         u.schema.Fields.ID,
			Type:         "any",
			IsNullable:   false,
			IsPrimaryKey: true,
			Tags: map[string]string{
				"json": "id",
			},
		},
		{
			Name:         u.schema.Fields.Email,
			Type:         "string",
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "email",
			},
		},
		{
			Name:         u.schema.Fields.Password,
			Type:         "string",
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "-",
			},
		},
		{
			Name:         u.schema.Fields.EmailVerifiedAt,
			Type:         "*time.Time",
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "email_verified_at",
			},
		},
	}

	if u.schema.Fields.SoftDeleteField != "" {
		fields = append(fields, FieldDefinition{
			Name:         u.schema.Fields.SoftDeleteField,
			Type:         "*time.Time",
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "deleted_at",
			},
		})
	}

	return fields
}

func (u *userSchemaIntrospector) GetIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:    "idx_users_email",
			Columns: []string{u.schema.Fields.Email},
			Unique:  true,
		},
	}
}

func (u *userSchemaIntrospector) GetForeignKeys() []ForeignKeyDefinition {
	return []ForeignKeyDefinition{}
}

func (u *userSchemaIntrospector) GetExtends() string {
	return ""
}
