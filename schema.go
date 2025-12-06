package aegis

import (
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type SchemaField string
type TableName string

// defaults for schemas table names
const (
	UserSchemaTableName         TableName = "users"
	VerificationSchemaTableName TableName = "verifications"
	SessionSchemaTableName      TableName = "sessions"
	RateLimitSchemaTableName    TableName = "rate_limits"
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

type Schema[T Model] interface {
	GetTableName() TableName
	ToStorage(data *T) map[string]any
	FromStorage(data map[string]any) *T
	GetSoftDeleteField() string
	GetAdditionalFields() AdditionalFieldsFunc
}

type Model interface {
	// Raw returns the model raw data as returned from the database
	Raw() map[string]any
}

func getNullableValue[T any](value any) *T {
	if value == nil {
		return nil
	}
	v := value.(T)
	return &v
}

func GetSchemaAdditionalFieldsForRequest[T Model](response http.ResponseWriter, request *http.Request, schema Schema[T]) (map[string]any, error) {
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
