package aegis

import (
	"time"
)

type User struct {
	ID              any        `json:"id"`
	Email           string     `json:"email"`
	Password        *string    `json:"-"`
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
		Password:        getNullableValue[string](data[u.GetPasswordField()]),
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
