package limen

import (
	"context"
	"net/http"
)

type contextKeyAdditionalFields struct{}

// Operation is the write an AdditionalFieldsFunc is being consulted for.
type Operation uint8

const (
	OperationCreate Operation = iota
	OperationUpdate
)

type AdditionalFieldsFunc func(ctx *AdditionalFieldsContext) (map[string]any, error)

type AdditionalFieldsContext struct {
	request   *http.Request
	response  http.ResponseWriter
	body      map[string]any
	schema    Schema
	operation Operation
}

func newAdditionalFieldsContext(request *http.Request, response http.ResponseWriter) *AdditionalFieldsContext {
	ctx := &AdditionalFieldsContext{
		request:  request,
		response: response,
		body:     GetJSONBody(request),
	}

	return ctx
}

func (ctx *AdditionalFieldsContext) GetBody() map[string]any {
	return ctx.body
}

func (ctx *AdditionalFieldsContext) GetBodyValue(key string) any {
	return ctx.body[key]
}

// Pick returns the body values for the given body key -> column name mapping,
// skipping the keys the request body omits. A key sent as null is kept, writing
// the column as NULL.
func (ctx *AdditionalFieldsContext) Pick(mapping map[string]string) map[string]any {
	fields := make(map[string]any, len(mapping))
	for key, column := range mapping {
		if value, exists := ctx.body[key]; exists {
			fields[column] = value
		}
	}
	return fields
}

func (ctx *AdditionalFieldsContext) GetHeader(key string) string {
	if ctx.request == nil {
		return ""
	}
	return ctx.request.Header.Get(key)
}

func (ctx *AdditionalFieldsContext) GetHeaders() http.Header {
	if ctx.request == nil {
		return nil
	}
	return ctx.request.Header
}

func (ctx *AdditionalFieldsContext) IsEmpty(key string) bool {
	return ctx.body[key] == nil || ctx.body[key] == ""
}

// Method returns the request method, empty outside an HTTP request.
func (ctx *AdditionalFieldsContext) Method() string {
	if ctx.request == nil {
		return ""
	}
	return ctx.request.Method
}

// Path returns the request path, empty outside an HTTP request.
func (ctx *AdditionalFieldsContext) Path() string {
	if ctx.request == nil || ctx.request.URL == nil {
		return ""
	}
	return ctx.request.URL.Path
}

func (ctx *AdditionalFieldsContext) Operation() Operation {
	return ctx.operation
}

func (ctx *AdditionalFieldsContext) IsCreate() bool {
	return ctx.operation == OperationCreate
}

func (ctx *AdditionalFieldsContext) IsUpdate() bool {
	return ctx.operation == OperationUpdate
}

// SchemaName returns the logical name of the schema being written.
func (ctx *AdditionalFieldsContext) SchemaName() SchemaName {
	if ctx.schema == nil {
		return ""
	}
	return ctx.schema.GetSchemaName()
}

// TableName returns the table being written.
func (ctx *AdditionalFieldsContext) TableName() SchemaTableName {
	if ctx.schema == nil {
		return ""
	}
	return ctx.schema.GetTableName()
}

func (ctx *AdditionalFieldsContext) forWrite(schema Schema, operation Operation) *AdditionalFieldsContext {
	ctx.schema = schema
	ctx.operation = operation
	return ctx
}

func withAdditionalFieldsContext(ctx context.Context, r *http.Request, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, contextKeyAdditionalFields{}, newAdditionalFieldsContext(r, w))
}

// getAdditionalFieldsContext retrieves the AdditionalFieldsContext from the req context.
// Returns an empty context (with nil request/response) if not in HTTP context (e.g., background jobs, CLI).
func getAdditionalFieldsContext(ctx context.Context) *AdditionalFieldsContext {
	if afCtx, ok := ctx.Value(contextKeyAdditionalFields{}).(*AdditionalFieldsContext); ok {
		return afCtx
	}

	return newAdditionalFieldsContext(nil, nil)
}
