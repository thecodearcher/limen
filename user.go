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

func (u *UserSchema) GetTableName() TableName {
	if u.TableName == "" {
		return UserSchemaTableName
	}
	return u.TableName
}

func (u *UserSchema) GetSoftDeleteField() string {
	return getFieldOrDefault(u.Fields.SoftDeleteField, "")
}

func (u *UserSchema) GetIDField() string {
	return getFieldOrDefault(u.Fields.ID, SchemaIDField)
}

func (u *UserSchema) GetEmailField() string {
	return getFieldOrDefault(u.Fields.Email, UserSchemaEmailField)
}

func (u *UserSchema) GetPasswordField() string {
	return getFieldOrDefault(u.Fields.Password, UserSchemaPasswordField)
}

func (u *UserSchema) GetEmailVerifiedAtField() string {
	return getFieldOrDefault(u.Fields.EmailVerifiedAt, UserSchemaEmailVerifiedAtField)
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
