package aegis

type SchemaDefinitionMap map[SchemaName]SchemaDefinition

type Schema interface {
	GetTableName() SchemaTableName
	ToStorage(data Model) map[string]any
	FromStorage(data map[string]any) Model
	GetSoftDeleteField() string
	GetAdditionalFields() AdditionalFieldsFunc
	GetIDField() string
	Initialize(schemaInfo *SchemaInfo) error
}

type Model interface {
	// Raw returns the model raw data as returned from the database
	Raw() map[string]any
}

type BaseSchema struct {
	// A function to return a map of additional fields to be added to the schema when creating a record. e.g:
	//  func(ctx context.Context) map[string]any {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }
	//	 }
	// NOTE: fields here will override the global additional fields function.
	additionalFields AdditionalFieldsFunc

	// schemaInfo contains all resolved schema information including table name, field mappings, and resolver
	schemaInfo *SchemaInfo
}

func (b *BaseSchema) GetTableName() SchemaTableName {
	if b.schemaInfo == nil {
		return ""
	}
	return b.schemaInfo.tableName
}

func (b *BaseSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return b.additionalFields
}

func (b *BaseSchema) GetIDField() string {
	return b.GetField(SchemaIDField)
}

func (b *BaseSchema) GetSoftDeleteField() string {
	return b.GetField(SchemaSoftDeleteField)
}

func (b *BaseSchema) GetFieldResolver() *SchemaResolver {
	if b.schemaInfo == nil {
		return nil
	}
	return b.schemaInfo.resolver
}

func (b *BaseSchema) GetField(name SchemaField) string {
	if b.schemaInfo == nil {
		return ""
	}
	return b.schemaInfo.GetField(name)
}

func (b *BaseSchema) Initialize(schemaInfo *SchemaInfo) error {
	b.schemaInfo = schemaInfo
	return nil
}
