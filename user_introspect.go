package aegis

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
