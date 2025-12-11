package aegis

import (
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type SchemaField string
type SchemaTableName string
type SchemaName string

// defaults for schemas table names
const (
	UserSchemaTableName         SchemaTableName = "users"
	VerificationSchemaTableName SchemaTableName = "verifications"
	SessionSchemaTableName      SchemaTableName = "sessions"
	RateLimitSchemaTableName    SchemaTableName = "rate_limits"
)

// defaults for schemas fields
const (
	SchemaIDField                  SchemaField = "id"
	UserSchemaEmailField           SchemaField = "email"
	UserSchemaPasswordField        SchemaField = "password"
	UserSchemaEmailVerifiedAtField SchemaField = "email_verified_at"
	UserSchemaCreatedAtField       SchemaField = "created_at"
	UserSchemaUpdatedAtField       SchemaField = "updated_at"
	UserSchemaSoftDeleteField      SchemaField = "deleted_at"

	VerificationSchemaSubjectField   SchemaField = "subject"
	VerificationSchemaValueField     SchemaField = "value"
	VerificationSchemaExpiresAtField SchemaField = "expires_at"
	VerificationSchemaCreatedAtField SchemaField = "created_at"
	VerificationSchemaUpdatedAtField SchemaField = "updated_at"

	SessionSchemaUserIDField     SchemaField = "user_id"
	SessionSchemaTokenField      SchemaField = "token"
	SessionSchemaCreatedAtField  SchemaField = "created_at"
	SessionSchemaExpiresAtField  SchemaField = "expires_at"
	SessionSchemaLastAccessField SchemaField = "last_access"
	SessionSchemaMetadataField   SchemaField = "metadata"

	RateLimitSchemaKeyField           SchemaField = "key"
	RateLimitSchemaCountField         SchemaField = "count"
	RateLimitSchemaLastRequestAtField SchemaField = "last_request_at"
)

type Schema interface {
	GetTableName() SchemaTableName
	ToStorage(data Model) map[string]any
	FromStorage(data map[string]any) Model
	GetSoftDeleteField() string
	GetAdditionalFields() AdditionalFieldsFunc
	Initialize(core *AegisCore, meta *PluginSchemaMetadata) error
}

type Model interface {
	// Raw returns the model raw data as returned from the database
	Raw() map[string]any
}

type BaseSchema struct {
	tableName SchemaTableName

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

	// mapping of the schema resolvedFields to the database columns
	resolvedFields map[string]string

	fieldResolver *FieldResolver

	meta *PluginSchemaMetadata
}

func NewBaseSchema(tableName SchemaTableName) *BaseSchema {
	return &BaseSchema{
		tableName: tableName,
	}
}

func (b *BaseSchema) GetTableName() SchemaTableName {
	return b.tableName
}

func (b *BaseSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return b.additionalFields
}

func (b *BaseSchema) GetSoftDeleteField() string {
	return ""
}

func (b *BaseSchema) GetFieldResolver() *FieldResolver {
	return b.fieldResolver
}

func (b *BaseSchema) GetField(name string) string {
	// if exists, ok := b.resolvedFields[name]; ok {
	// 	return exists
	// }
	// return name
	if b.meta == nil {
		return name
	}
	if field, err := b.meta.GetField(name); err == nil {
		return field
	}
	return name
}

func (b *BaseSchema) Initialize(core *AegisCore, meta *PluginSchemaMetadata) error {
	b.fieldResolver = meta.FieldResolver
	b.resolvedFields = meta.Fields
	b.tableName = meta.TableName
	b.meta = meta
	return nil
}

type AdditionalFieldsFunc func(ctx *AdditionalFieldsContext) (map[string]any, *AegisError)

type AdditionalFieldsContext struct {
	request  *http.Request
	response http.ResponseWriter
	body     map[string]any
}

// GetBody returns the body of the request if it exists
func (ctx *AdditionalFieldsContext) GetBody() map[string]any {
	return ctx.body
}

// GetBodyValue returns the value of the request body for the given key
func (ctx *AdditionalFieldsContext) GetBodyValue(key string) any {
	return ctx.body[key]
}

// GetHeader returns the value of the request header for the given key
func (ctx *AdditionalFieldsContext) GetHeader(key string) string {
	return ctx.request.Header.Get(key)
}

// GetHeaders returns the headers of the request
func (ctx *AdditionalFieldsContext) GetHeaders() http.Header {
	return ctx.request.Header
}

// NewAdditionalFieldsContext creates a new additional fields context
func NewAdditionalFieldsContext(request *http.Request, response http.ResponseWriter) *AdditionalFieldsContext {
	ctx := &AdditionalFieldsContext{
		request:  request,
		response: response,
		body:     httpx.GetJSONBody(request),
	}

	return ctx
}

func getNullableValue[T any](value any) *T {
	if value == nil {
		return nil
	}
	v := value.(T)
	return &v
}

func GetSchemaAdditionalFieldsForRequest(response http.ResponseWriter, request *http.Request, schema Schema) (map[string]any, error) {
	additionalFieldsContext := NewAdditionalFieldsContext(request, response)
	if schema.GetAdditionalFields() != nil {
		value, err := schema.GetAdditionalFields()(additionalFieldsContext)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
	return make(map[string]any), nil
}
