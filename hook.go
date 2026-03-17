package limen

import (
	"net/http"
	"strings"
)

type HookFunc func(ctx *HookContext) bool
type PathMatcherFunc func(ctx *HookContext) bool

// Hook is a function that runs before or after a request and can optionally restrict which requests it runs for.
type Hook struct {
	// Run is the function to execute for the hook. It can return false to stop the request from continuing.
	Run HookFunc
	// PathMatcher is a function that returns whether the hook should run for the given context.
	PathMatcher PathMatcherFunc
}

// Hooks is a container for optional before and after hooks to add to the router.
// Hooks in Before run in order before the request handler; hooks in After run in order after.
// Any before-hook returning false stops the chain and the request does not continue.
type Hooks struct {
	Before []*Hook
	After  []*Hook
}

// ResponseData represents the response data that hooks can read and modify
type ResponseData struct {
	StatusCode int
	Payload    any
	IsError    bool
	Headers    http.Header
}

type HookContext struct {
	request          *http.Request
	response         http.ResponseWriter
	routeID          string
	routePattern     string
	method           string
	path             string
	statusCode       int
	originalBodyData map[string]any
	modifiedData     map[string]any
	bodyModified     bool
	responder        *Responder
}

// Getter methods for read-only access to HookContext fields

func (hc *HookContext) Request() *http.Request          { return hc.request }
func (hc *HookContext) Response() http.ResponseWriter   { return hc.response }
func (hc *HookContext) RouteID() string                 { return hc.routeID }
func (hc *HookContext) RoutePattern() string            { return hc.routePattern }
func (hc *HookContext) Method() string                  { return hc.method }
func (hc *HookContext) Path() string                    { return hc.path }
func (hc *HookContext) StatusCode() int                 { return hc.statusCode }
func (hc *HookContext) GetJSONBodyData() map[string]any { return hc.originalBodyData }
func (hc *HookContext) GetJSONBodyValue(key string) any { return hc.originalBodyData[key] }

// WriteResponse writes a response to the client and should only be used in a before hook
// when you want to return a response immediately without waiting for the request to complete.
func (hc *HookContext) WriteJSONResponse(status int, payload any) {
	hc.responder.JSON(hc.response, hc.request, status, payload)
}

// WriteErrorResponse writes an error response to the client and should only be used in a before hook
// when you want to return an error response immediately without waiting for the request to complete.
func (hc *HookContext) WriteErrorResponse(err error) {
	hc.responder.Error(hc.response, hc.request, err)
}

func (hc *HookContext) SetBody(data map[string]any) {
	hc.modifiedData = data
	hc.bodyModified = true
}

// GetResponse returns the current response data if available, nil otherwise
func (hc *HookContext) GetResponse() *ResponseData {
	rw, ok := hc.response.(*responseWriter)
	if !ok || !rw.written {
		return nil
	}
	return &ResponseData{
		StatusCode: rw.statusCode,
		Payload:    rw.payload,
		IsError:    rw.isError,
		Headers:    rw.Header(),
	}
}

// GetAuthResult returns the AuthenticationResult stored during SessionResponse, if available
func (hc *HookContext) GetAuthResult() *AuthenticationResult {
	rw, ok := hc.response.(*responseWriter)
	if !ok {
		return nil
	}
	return rw.authResult
}

// ModifyResponse allows hooks to modify the response payload and status code
func (hc *HookContext) ModifyResponse(status int, payload any) {
	rw, ok := hc.response.(*responseWriter)
	if !ok {
		return
	}
	rw.modified = true
	rw.modifiedStatus = status
	rw.modifiedPayload = payload
}

// SetResponseHeader sets a response header
func (hc *HookContext) SetResponseHeader(key, value string) {
	hc.response.Header().Set(key, value)
}

// SetResponseCookie adds a cookie to the response
func (hc *HookContext) SetResponseCookie(cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	http.SetCookie(hc.response, cookie)
}

// DeleteResponseHeader removes a header from the response entirely
func (hc *HookContext) DeleteResponseHeader(key string) {
	hc.response.Header().Del(key)
}

// RemoveResponseCookie removes a specific cookie from the response headers,
// preventing it from being sent to the client. This is different from DeleteResponseCookie
// which sends a Set-Cookie header telling the browser to delete the cookie.
func (hc *HookContext) RemoveResponseCookie(name string) {
	h := hc.response.Header()
	cookies := h.Values("Set-Cookie")
	h.Del("Set-Cookie")
	for _, c := range cookies {
		// Cookie format is "name=value; ..." - check if it starts with the target name
		if !strings.HasPrefix(c, name+"=") {
			h.Add("Set-Cookie", c)
		}
	}
}
