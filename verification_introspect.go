package aegis

func (v *VerificationSchema) Introspect(config *SchemaConfig) SchemaIntrospector {
	fields := v.getDefaultColumns(config)

	return &SchemaDefinition{
		TableName: VerificationSchemaTableName,
		Columns:   fields,
		Indexes: []IndexDefinition{
			{
				Name:    "idx_verifications_value",
				Columns: []SchemaField{VerificationSchemaValueField},
				Unique:  true,
			},
			{
				Name:    "idx_verifications_subject",
				Columns: []SchemaField{VerificationSchemaSubjectField},
				Unique:  false,
			},
		},
		SchemaName: CoreSchemaVerifications,
		Schema:     v,
	}
}

func (v *VerificationSchema) getDefaultColumns(config *SchemaConfig) []ColumnDefinition {
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
			Name:         string(VerificationSchemaSubjectField),
			LogicalField: VerificationSchemaSubjectField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "subject",
			},
		},
		{
			Name:         string(VerificationSchemaValueField),
			LogicalField: VerificationSchemaValueField,
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "value",
			},
		},
		{
			Name:         string(VerificationSchemaExpiresAtField),
			LogicalField: VerificationSchemaExpiresAtField,
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "expires_at",
			},
		},
	}

	fields = addTimestampFields(fields)

	fields = addSoftDeleteField(fields, config, CoreSchemaVerifications)

	return fields
}
