package twofactor

import "github.com/thecodearcher/limen"

type SchemaUserTwoFactorOption func(*twoFactorSchema)

type userWithTwoFactorSchema struct {
	*limen.UserSchema
}

type UserWithTwoFactor struct {
	*limen.User
	TwoFactorEnabled bool
}

func newDefaultSchemaUserTwoFactor(userSchema *limen.UserSchema, opts ...SchemaUserTwoFactorOption) *userWithTwoFactorSchema {
	return &userWithTwoFactorSchema{
		UserSchema: userSchema,
	}
}

func (u *userWithTwoFactorSchema) GetTwoFactorEnabledField() string {
	return u.GetField(UserWithTwoFactorSchemaEnabledField)
}

func (u *userWithTwoFactorSchema) FromStorage(data map[string]any) limen.Model {
	user := u.UserSchema.FromStorage(data)
	return &UserWithTwoFactor{
		User:             user.(*limen.User),
		TwoFactorEnabled: data[u.GetTwoFactorEnabledField()].(bool),
	}
}

func (u *userWithTwoFactorSchema) ToStorage(data limen.Model) map[string]any {
	user := data.(*UserWithTwoFactor)
	result := make(map[string]any)
	if user.User != nil {
		result = u.UserSchema.ToStorage(user.User)
	}
	result[u.GetTwoFactorEnabledField()] = user.TwoFactorEnabled
	return result
}

func (t *userWithTwoFactorSchema) UserToUserWithTwoFactor(user *limen.User) *UserWithTwoFactor {
	raw := user.Raw()
	return &UserWithTwoFactor{
		User:             user,
		TwoFactorEnabled: raw[string(t.GetTwoFactorEnabledField())].(bool),
	}
}
