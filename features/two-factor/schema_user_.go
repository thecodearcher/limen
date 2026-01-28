package twofactor

import "github.com/thecodearcher/aegis"

type SchemaUserTwoFactorOption func(*twoFactorSchema)

type userWithTwoFactorSchema struct {
	*aegis.UserSchema
}

type UserWithTwoFactor struct {
	*aegis.User
	TwoFactorEnabled bool
}

func newDefaultSchemaUserTwoFactor(userSchema *aegis.UserSchema, opts ...SchemaUserTwoFactorOption) *userWithTwoFactorSchema {
	return &userWithTwoFactorSchema{
		UserSchema: userSchema,
	}
}

func (u *userWithTwoFactorSchema) GetTwoFactorEnabledField() string {
	return u.GetField(UserWithTwoFactorSchemaEnabledField)
}

func (u *userWithTwoFactorSchema) FromStorage(data map[string]any) aegis.Model {
	user := u.UserSchema.FromStorage(data)
	return &UserWithTwoFactor{
		User:             user.(*aegis.User),
		TwoFactorEnabled: data[u.GetTwoFactorEnabledField()].(bool),
	}
}

func (u *userWithTwoFactorSchema) ToStorage(data aegis.Model) map[string]any {
	user := data.(*UserWithTwoFactor)
	result := make(map[string]any)
	if user.User != nil {
		result = u.UserSchema.ToStorage(user.User)
	}
	result[u.GetTwoFactorEnabledField()] = user.TwoFactorEnabled
	return result
}

func (t *userWithTwoFactorSchema) UserToUserWithTwoFactor(user *aegis.User) *UserWithTwoFactor {
	raw := user.Raw()
	return &UserWithTwoFactor{
		User:             user,
		TwoFactorEnabled: raw[string(t.GetTwoFactorEnabledField())].(bool),
	}
}
