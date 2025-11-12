package httpx

import (
	"context"
	"net/http"
)

type HookFunc func(ctx *HookContext) bool

type Hooks struct {
	Before HookFunc
	After  HookFunc
}

type HookContext struct {
	Context      context.Context
	Request      *http.Request
	Response     http.ResponseWriter
	RouteID      string
	RoutePattern string
	Method       string
	Path         string
	modifiedData map[string]any
	StatusCode   int

	bodyModified bool
}

func (hc *HookContext) SetBody(data map[string]any) {
	hc.modifiedData = data
	hc.bodyModified = true
}

// responseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
