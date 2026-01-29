package aegis

import (
	"context"
	"net/http"
)

type contextKeyAdditionalFields struct{}

type AdditionalFieldsFunc func(ctx *AdditionalFieldsContext) (map[string]any, error)

type AdditionalFieldsContext struct {
	request  *http.Request
	response http.ResponseWriter
	body     map[string]any
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

func (ctx *AdditionalFieldsContext) GetHeader(key string) string {
	return ctx.request.Header.Get(key)
}

func (ctx *AdditionalFieldsContext) GetHeaders() http.Header {
	return ctx.request.Header
}

func (ctx *AdditionalFieldsContext) IsEmpty(key string) bool {
	return ctx.body[key] == nil || ctx.body[key] == ""
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
