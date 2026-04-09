package limen

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"path"
	"slices"
	"strings"
)

type paramsContextKey struct{}
type currentRouteContextKey struct{}

type HTTPMethod string

const (
	MethodANY     HTTPMethod = "ANY"
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodPATCH   HTTPMethod = "PATCH"
	MethodHEAD    HTTPMethod = "HEAD"
	MethodOPTIONS HTTPMethod = "OPTIONS"
)

// methodIndex maps HTTP methods to array indices
var methodIndex = map[HTTPMethod]int{
	MethodGET:     0,
	MethodPOST:    1,
	MethodPUT:     2,
	MethodDELETE:  3,
	MethodPATCH:   4,
	MethodHEAD:    5,
	MethodOPTIONS: 6,
	MethodANY:     7,
}

type Middleware func(http.Handler) http.Handler

// RadixNode represents a node in the radix tree
type RadixNode struct {
	path string

	routes [8]*Route

	children map[string]*RadixNode

	// Parameter child (for :param routes)
	paramChild *RadixNode
	paramName  string

	// Whether this node represents a parameter (starts with :)
	isParam bool
}

// Router is a radix tree-based HTTP router optimized for authentication endpoints
// Supports:
// - Static segments
// - :param (single segment parameters)
// - HEAD -> GET fallback
type Router struct {
	root             *RadixNode
	globalMiddleware []Middleware
	beforeHooks      []Hook
	afterHooks       []Hook
	responder        *Responder // For final response writing after hooks
}

type RouteMetadata struct {
	AllowedContentTypes []string
	// originalPattern is the original pattern of the route before any normalization or prefixing
	originalPattern string
}

// Route represents a single route with its handler and metadata
type Route struct {
	Method      HTTPMethod
	Pattern     string
	Handler     http.HandlerFunc
	RouteID     RouteID
	Description string
	Middleware  []Middleware
	Metadata    *RouteMetadata
}

// RouteID is a unique identifier for each route
type RouteID string

// RouterGroup represents a group of routes with a common prefix and middleware.
// Routes added to a group automatically have the prefix prepended and group middleware applied.
type RouterGroup struct {
	router     *Router
	prefix     string
	middleware []Middleware
}

// NewRouter creates a new radix tree router instance.
// Add global or plugin hooks with AddHooks.
func NewRouter(responder *Responder, globalMiddleware ...Middleware) *Router {
	return &Router{
		root: &RadixNode{
			children: make(map[string]*RadixNode),
		},
		globalMiddleware: globalMiddleware,
		responder:        responder,
	}
}

// AddHooks appends the hook set's Before and After hooks to the router.
func (r *Router) AddHooks(h *Hooks) {
	if h == nil {
		return
	}
	for _, hook := range h.Before {
		if hook != nil {
			r.beforeHooks = append(r.beforeHooks, *hook)
		}
	}
	for _, hook := range h.After {
		if hook != nil {
			r.afterHooks = append(r.afterHooks, *hook)
		}
	}
}

// AddRoute adds a new route to the radix tree.
// Middleware is applied in order: global middleware first, then route-specific middleware.
func (r *Router) AddRoute(method HTTPMethod, pattern string, handler http.HandlerFunc, routeID RouteID, metadata *RouteMetadata, middleware ...Middleware) {
	route := &Route{
		Method:     method,
		Pattern:    pattern,
		Handler:    handler,
		RouteID:    routeID,
		Middleware: middleware,
		Metadata:   metadata,
	}

	segments := r.splitPath(pattern)
	r.insertRoute(route, segments)
}

// Group creates a new router group with the given prefix and middleware.
// All routes added to the group will have the prefix prepended to their paths.
func (r *Router) Group(prefix string, middleware ...Middleware) *RouterGroup {
	prefix = NormalizePath(prefix)
	return &RouterGroup{
		router:     r,
		prefix:     prefix,
		middleware: middleware,
	}
}

// insertRoute iteratively inserts a route into the radix tree
func (r *Router) insertRoute(route *Route, segments []string) {
	current := r.root

	for _, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			current = r.handleParameterSegment(current, segment)
			continue
		}

		current = r.handleStaticSegment(current, segment)
	}

	methodIdx := methodIndex[route.Method]
	current.routes[methodIdx] = route
}

// handleParameterSegment handles parameter segments with early returns
func (r *Router) handleParameterSegment(current *RadixNode, segment string) *RadixNode {
	paramName := segment[1:]

	if current.paramChild != nil {
		if current.paramChild.paramName != paramName {
			panic("conflicting parameter names at same path level")
		}
		return current.paramChild
	}

	current.paramChild = &RadixNode{
		path:      segment,
		children:  make(map[string]*RadixNode),
		isParam:   true,
		paramName: paramName,
	}
	return current.paramChild
}

// handleStaticSegment handles static segments with early returns
func (r *Router) handleStaticSegment(current *RadixNode, segment string) *RadixNode {
	if child, exists := current.children[segment]; exists {
		return child
	}

	child := &RadixNode{
		path:     segment,
		children: make(map[string]*RadixNode),
	}
	current.children[segment] = child
	return child
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	segments := r.splitPath(req.URL.Path)
	route, params := r.matchRoute(segments, HTTPMethod(req.Method))
	if route != nil {
		r.handleRoute(w, req, route, params)
		return
	}

	http.NotFound(w, req)
}

// wrapHandler applies global middleware, route-specific middleware to a handler
// and applies hooks to the request and response and this is where the request body is parsed and stored in the request context
func (r *Router) wrapHandler(handler http.HandlerFunc, routeMiddleware []Middleware, route *Route) http.HandlerFunc {
	allMiddleware := slices.Concat(r.globalMiddleware, routeMiddleware)
	wrapped := r.applyMiddleware(allMiddleware, handler)
	hasAfterHooks := len(r.afterHooks) > 0

	return func(w http.ResponseWriter, req *http.Request) {
		req = parseAndStoreBody(req)

		hookCtx := r.prepareHookContext(req, w, route)
		if !r.runBeforeHooks(hookCtx) {
			return
		}

		if hookCtx.bodyModified {
			bodyBytes, _ := json.Marshal(hookCtx.modifiedData)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			req = req.WithContext(context.WithValue(req.Context(), bodyContextKey{}, hookCtx.modifiedData))
			hookCtx.request = req
		}

		rw := &responseWriter{
			ResponseWriter: w,
			wroteHeader:    false,
			deferWrite:     hasAfterHooks,
		}
		hookCtx.response = rw

		wrapped.ServeHTTP(rw, req)

		// Only run after hooks logic if we have them
		if hasAfterHooks {
			hookCtx.statusCode = rw.statusCode
			r.runAfterHooks(hookCtx)

			r.writeFinalResponse(rw, req)
		}
	}
}

// writeFinalResponse writes the final response after hooks have run
func (r *Router) writeFinalResponse(rw *responseWriter, req *http.Request) {
	if !rw.written || r.responder == nil {
		return // Handler didn't use Responder, response already sent
	}

	// Deferred redirect: send it on the real ResponseWriter so the browser follows it
	if rw.redirectURL != "" {
		http.Redirect(rw.ResponseWriter, req, rw.redirectURL, rw.redirectStatus)
		return
	}

	status := rw.statusCode
	payload := rw.payload
	isError := rw.isError

	if rw.modified {
		status = rw.modifiedStatus
		payload = rw.modifiedPayload
		isError = false // Modified responses are treated as success and rely on error type payload
	}

	if err, ok := payload.(error); ok || isError {
		r.responder.Error(rw.ResponseWriter, req, err)
		return
	}

	r.responder.JSON(rw.ResponseWriter, req, status, payload)
}

func (r *Router) prepareHookContext(req *http.Request, w http.ResponseWriter, route *Route) *HookContext {
	routePattern := ""
	if route.Metadata != nil {
		routePattern = route.Metadata.originalPattern
	}
	return &HookContext{
		responder:        r.responder,
		request:          req,
		response:         w,
		method:           req.Method,
		path:             req.URL.Path,
		routeID:          string(route.RouteID),
		routePattern:     routePattern,
		originalBodyData: GetJSONBody(req),
	}
}

func (r *Router) runBeforeHooks(hookCtx *HookContext) bool {
	for _, hook := range r.beforeHooks {
		if hook.PathMatcher == nil || hook.PathMatcher(hookCtx) {
			if !hook.Run(hookCtx) {
				return false
			}
		}
	}
	return true
}

func (r *Router) runAfterHooks(hookCtx *HookContext) bool {
	for _, hook := range r.afterHooks {
		if hook.PathMatcher == nil || hook.PathMatcher(hookCtx) {
			if !hook.Run(hookCtx) {
				return false
			}
		}
	}
	return true
}

// handleRoute handles a matched route with parameters
func (r *Router) handleRoute(w http.ResponseWriter, req *http.Request, route *Route, params map[string]string) {
	ctx := context.WithValue(req.Context(), currentRouteContextKey{}, route)
	req = req.WithContext(ctx)

	if len(params) > 0 {
		ctx := context.WithValue(req.Context(), paramsContextKey{}, params)
		req = req.WithContext(ctx)
	}

	wrappedHandler := r.wrapHandler(route.Handler, route.Middleware, route)
	wrappedHandler(w, req)
}

// matchRoute searches the radix tree for a matching route
func (r *Router) matchRoute(segments []string, method HTTPMethod) (*Route, map[string]string) {
	current := r.root
	params := make(map[string]string)
	methodIdx := methodIndex[method]
	// track nearest ANY prefix
	var lastAny *Route
	lastAnyParams := map[string]string{}

	// check root for ANY (if you ever mount at "/")
	if rt := current.routes[methodIndex[MethodANY]]; rt != nil {
		lastAny, lastAnyParams = rt, copyParams(params)
	}

	for _, segment := range segments {
		if child, exists := current.children[segment]; exists {
			current = child
			if rt := current.routes[methodIndex[MethodANY]]; rt != nil {
				lastAny, lastAnyParams = rt, copyParams(params)
			}
			continue
		}

		if current.paramChild != nil {
			current = current.paramChild
			params[current.paramName] = segment
			if rt := current.routes[methodIndex[MethodANY]]; rt != nil {
				lastAny, lastAnyParams = rt, copyParams(params)
			}
			continue
		}

		// failed deeper; use nearest prefix ANY if available
		if lastAny != nil {
			return lastAny, lastAnyParams
		}
		return nil, nil
	}

	if route := current.routes[methodIdx]; route != nil {
		return route, params
	}

	if route := current.routes[methodIndex[MethodANY]]; route != nil {
		return route, params
	}

	// path fully consumed but no handler; try nearest prefix ANY
	if lastAny != nil {
		return lastAny, lastAnyParams
	}
	return nil, nil
}

func copyParams(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m))
	maps.Copy(cp, m)
	return cp
}

// splitPath splits a path into segments, removing empty segments
func (r *Router) splitPath(pathStr string) []string {
	pathStr = path.Clean(pathStr)

	if pathStr == "/" || pathStr == "" {
		return []string{}
	}

	pathStr = strings.TrimPrefix(pathStr, "/")
	return strings.Split(pathStr, "/")
}

// AddRoute adds a route to the group with the group's prefix prepended.
// Middleware is applied in order: router global middleware, group middleware, then route-specific middleware.
func (g *RouterGroup) AddRoute(method HTTPMethod, pattern string, handler http.HandlerFunc, routeID RouteID, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := slices.Concat(g.middleware, middleware)
	fullPattern := g.prefix + NormalizePath(pattern)
	if metadata == nil {
		metadata = &RouteMetadata{}
	}
	metadata.originalPattern = pattern
	g.router.AddRoute(method, fullPattern, handler, routeID, metadata, allMiddleware...)
}

func (r *Router) applyMiddleware(mws []Middleware, h http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		if mw := mws[i]; mw != nil {
			h = mw(h)
		}
	}
	return h
}

// GetParams extracts parameters from the request context
func GetParams(req *http.Request) map[string]string {
	if params, ok := req.Context().Value(paramsContextKey{}).(map[string]string); ok {
		return params
	}
	return make(map[string]string)
}

// GetParam extracts a specific parameter from the request context
func GetParam(req *http.Request, name string) string {
	params := GetParams(req)
	return params[name]
}
