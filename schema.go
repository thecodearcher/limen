package aegis

type Schema[T Model] interface {
	GetTableName() SchemaTableName
	// ToStorage() any
	FromStorage(data map[string]any) T
}

type UserSchema struct {
	// name of the table in the database
	TableName SchemaTableName
	// mapping of the user schema to the database columns
	Fields UserFields
}

type Model interface {
	Name() string
}

type User struct {
	ID        any
	FirstName string
	LastName  string
	Email     string
	Password  string
}

type UserFields struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Password  string
}

func (c User) Name() string {
	return string(UserSchemaTableName)
}

func (c *UserSchema) GetTableName() SchemaTableName {
	if c.TableName == "" {
		return UserSchemaTableName
	}
	return c.TableName
}

func (c *UserSchema) GetIDField() string {
	if c.Fields.ID == "" {
		return string(SchemaIDField)
	}
	return c.Fields.ID
}

func (c *UserSchema) GetEmailField() string {
	if c.Fields.Email == "" {
		return string(UserSchemaEmailField)
	}
	return c.Fields.Email
}

func (c *UserSchema) GetFirstNameField() string {
	if c.Fields.FirstName == "" {
		return string(UserSchemaFirstNameField)
	}
	return c.Fields.FirstName
}

func (c *UserSchema) GetLastNameField() string {
	if c.Fields.LastName == "" {
		return string(UserSchemaLastNameField)
	}
	return c.Fields.LastName
}

func (c *UserSchema) GetPasswordField() string {
	if c.Fields.Password == "" {
		return string(UserSchemaPasswordField)
	}
	return c.Fields.Password
}

func (c *UserSchema) FromStorage(data map[string]any) User {
	return User{
		ID:        data[c.GetIDField()],
		Email:     data[c.GetEmailField()].(string),
		FirstName: data[c.GetFirstNameField()].(string),
		LastName:  data[c.GetLastNameField()].(string),
		Password:  data[c.GetPasswordField()].(string),
	}
}
