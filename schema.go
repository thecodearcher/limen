package limen

import (
	"context"
	"maps"
)

type SchemaDefinitionMap map[SchemaName]SchemaDefinition

type Schema interface {
	GetSchemaName() SchemaName
	GetTableName() SchemaTableName
	GetField(name SchemaField) string
	ToStorage(data Model) map[string]any
	FromStorage(data map[string]any) Model
	Serialize(data Model) map[string]any
	GetSoftDeleteField() string
	GetAdditionalFields() AdditionalFieldsFunc
	GetIDField() string
	Initialize(schemaInfo *SchemaInfo) error
	setAdditionalFields(additionalFields AdditionalFieldsFunc)
	setModelTransformer(transformer ModelTransformer)
}

type Model interface {
	// Raw returns the model raw data as returned from the database
	Raw() map[string]any
}

// ModelTransformer transforms a model into its JSON response representation.
type ModelTransformer func(model Model) map[string]any

// ModelTransformers maps logical schema names to model transformers.
type ModelTransformers map[SchemaName]ModelTransformer

type PublicIDGenerator func(ctx context.Context, schemaName SchemaName) (string, error)
type PublicIDMatcher func(schemaName SchemaName, value string) bool
type PublicIDEncoder func(schemaName SchemaName, value string) string
type PublicIDDecoder func(schemaName SchemaName, publicID string) (string, error)

type PublicIDConfig struct {
	Disabled bool
	// The schemas that the public-ID is disabled for
	DisabledFor []SchemaName
	// The logical field name of the public-ID field
	field SchemaField
	// The database column name of the public-ID field
	ColumnName string
	// The database column type of the public-ID field
	ColumnType ColumnType
	// Generator produces the stored public-ID value on insert. Optional: when nil,
	// generation is skipped and the value is expected from additionalFields or a
	// database default.
	Generator PublicIDGenerator
	// Matcher decides which values route to the public-ID column. Required.
	Matcher PublicIDMatcher
	// Encoder transforms the stored value into the outward-facing ID. Optional:
	// defaults to identity, so the field is exposed as-is.
	Encoder PublicIDEncoder
	// Decoder transforms an incoming ID back into the stored value. Optional:
	// defaults to identity, so the field is queried as-is.
	Decoder PublicIDDecoder
	// The field name of the json response that will be returned to the client
	ResponseField string
	// If true, the response transform will be disabled
	DisableResponseTransform bool
}

type BaseSchema struct {
	// A function to return a map of additional fields to be added to the schema when writing a record. e.g:
	//  func(ctx *AdditionalFieldsContext) (map[string]any, error) {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }, nil
	//	 }
	// NOTE: fields here will override the global additional fields function, and the function
	// runs on both creates and updates.
	additionalFields AdditionalFieldsFunc

	// schemaInfo contains all resolved schema information including table name, field mappings, and resolver
	schemaInfo *SchemaInfo

	// A function to serialize the model to a json object for returning to the client
	Serializer ModelTransformer
}

func (b *BaseSchema) GetSchemaName() SchemaName {
	if b.schemaInfo == nil {
		return ""
	}
	return b.schemaInfo.schemaName
}

func (b *BaseSchema) GetTableName() SchemaTableName {
	if b.schemaInfo == nil {
		return ""
	}
	return b.schemaInfo.tableName
}

func (b *BaseSchema) setAdditionalFields(additionalFields AdditionalFieldsFunc) {
	b.additionalFields = additionalFields
}

func (b *BaseSchema) setModelTransformer(transformer ModelTransformer) {
	b.Serializer = transformer
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

func (b *BaseSchema) Serialize(data Model) map[string]any {
	if b.Serializer != nil {
		return b.Serializer(data)
	}
	return maps.Clone(data.Raw())
}

func (b *BaseSchema) Initialize(schemaInfo *SchemaInfo) error {
	b.schemaInfo = schemaInfo
	return nil
}
