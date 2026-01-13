package aegis

import (
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type AdditionalFieldsFunc func(ctx *AdditionalFieldsContext) (map[string]any, *AegisError)

type AdditionalFieldsContext struct {
	request  *http.Request
	response http.ResponseWriter
	body     map[string]any
}

func newAdditionalFieldsContext(request *http.Request, response http.ResponseWriter) *AdditionalFieldsContext {
	ctx := &AdditionalFieldsContext{
		request:  request,
		response: response,
		body:     httpx.GetJSONBody(request),
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
