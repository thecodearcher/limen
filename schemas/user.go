package schemas

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
	// field name for the soft delete field - if not set, the soft delete field will not be used
	SoftDeleteField SchemaField
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
	FirstName       string
	LastName        string
	Email           string
	Password        string
	EmailVerifiedAt string
}

func (c *UserSchema) GetTableName() TableName {
	if c.TableName == "" {
		return UserSchemaTableName
	}
	return c.TableName
}

func (c *UserSchema) GetSoftDeleteField() SchemaField {
	return c.SoftDeleteField
}

func (c *UserSchema) GetIDField() string {
	return getFieldOrDefault(c.Fields.ID, SchemaIDField)
}

func (c *UserSchema) GetEmailField() string {
	return getFieldOrDefault(c.Fields.Email, UserSchemaEmailField)
}

func (c *UserSchema) GetPasswordField() string {
	return getFieldOrDefault(c.Fields.Password, UserSchemaPasswordField)
}

func (c *UserSchema) GetEmailVerifiedAtField() string {
	return getFieldOrDefault(c.Fields.EmailVerifiedAt, UserSchemaEmailVerifiedAtField)
}

func (c *UserSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return c.AdditionalFields
}

func (c *UserSchema) FromStorage(data map[string]any) *User {
	return &User{
		ID:              data[c.GetIDField()],
		Email:           data[c.GetEmailField()].(string),
		Password:        data[c.GetPasswordField()].(string),
		EmailVerifiedAt: getNullableValue[time.Time](data[c.GetEmailVerifiedAtField()]),
		raw:             data,
	}
}

func (c *UserSchema) ToStorage(data *User) map[string]any {
	return map[string]any{
		c.GetEmailField():           data.Email,
		c.GetPasswordField():        data.Password,
		c.GetEmailVerifiedAtField(): data.EmailVerifiedAt,
	}
}

func (c *UserSchema) Serialize(data *User) map[string]any {
	if c.Serializer != nil {
		return c.Serializer(data)
	}
	raw := data.Raw()
	delete(raw, c.GetPasswordField())
	return raw
}
