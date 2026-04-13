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

// radixNode is a node in the radix tree.
type radixNode struct {
	path string

	routes [8]*route

	children map[string]*radixNode

	// Parameter child (for :param routes)
	paramChild *radixNode
	paramName  string

	// Whether this node represents a parameter (starts with :)
	isParam bool
}

// router is a radix tree-based HTTP router optimized for authentication endpoints.
// Supports static segments, :param (single segment parameters), and HEAD -> GET fallback.
type router struct {
	root             *radixNode
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

// route is a single registered route with its handler and metadata.
type route struct {
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

// routerGroup is a group of routes with a common prefix and middleware.
// Routes added to a group automatically have the prefix prepended and group middleware applied.
type routerGroup struct {
	router     *router
	prefix     string
	middleware []Middleware
}

// newRouter creates a new radix tree router instance.
// Add global or plugin hooks with AddHooks.
func newRouter(responder *Responder, globalMiddleware ...Middleware) *router {
	return &router{
		root: &radixNode{
			children: make(map[string]*radixNode),
		},
		globalMiddleware: globalMiddleware,
		responder:        responder,
	}
}

// AddHooks appends the hook set's Before and After hooks to the router.
func (r *router) AddHooks(h *Hooks) {
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
func (r *router) AddRoute(method HTTPMethod, pattern string, handler http.HandlerFunc, routeID RouteID, metadata *RouteMetadata, middleware ...Middleware) {
	route := &route{
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
func (r *router) Group(prefix string, middleware ...Middleware) *routerGroup {
	prefix = normalizePath(prefix)
	return &routerGroup{
		router:     r,
		prefix:     prefix,
		middleware: middleware,
	}
}

// insertRoute iteratively inserts a route into the radix tree
func (r *router) insertRoute(route *route, segments []string) {
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
func (r *router) handleParameterSegment(current *radixNode, segment string) *radixNode {
	paramName := segment[1:]

	if current.paramChild != nil {
		if current.paramChild.paramName != paramName {
			panic("conflicting parameter names at same path level")
		}
		return current.paramChild
	}

	current.paramChild = &radixNode{
		path:      segment,
		children:  make(map[string]*radixNode),
		isParam:   true,
		paramName: paramName,
	}
	return current.paramChild
}

// handleStaticSegment handles static segments with early returns
func (r *router) handleStaticSegment(current *radixNode, segment string) *radixNode {
	if child, exists := current.children[segment]; exists {
		return child
	}

	child := &radixNode{
		path:     segment,
		children: make(map[string]*radixNode),
	}
	current.children[segment] = child
	return child
}

// ServeHTTP implements http.Handler
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
func (r *router) wrapHandler(handler http.HandlerFunc, routeMiddleware []Middleware, route *route) http.HandlerFunc {
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
func (r *router) writeFinalResponse(rw *responseWriter, req *http.Request) {
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

func (r *router) prepareHookContext(req *http.Request, w http.ResponseWriter, route *route) *HookContext {
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

func (r *router) runBeforeHooks(hookCtx *HookContext) bool {
	for _, hook := range r.beforeHooks {
		if hook.PathMatcher == nil || hook.PathMatcher(hookCtx) {
			if !hook.Run(hookCtx) {
				return false
			}
		}
	}
	return true
}

func (r *router) runAfterHooks(hookCtx *HookContext) bool {
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
func (r *router) handleRoute(w http.ResponseWriter, req *http.Request, route *route, params map[string]string) {
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
func (r *router) matchRoute(segments []string, method HTTPMethod) (*route, map[string]string) {
	current := r.root
	params := make(map[string]string)
	methodIdx := methodIndex[method]
	// track nearest ANY prefix
	var lastAny *route
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
func (r *router) splitPath(pathStr string) []string {
	pathStr = path.Clean(pathStr)

	if pathStr == "/" || pathStr == "" {
		return []string{}
	}

	pathStr = strings.TrimPrefix(pathStr, "/")
	return strings.Split(pathStr, "/")
}

// AddRoute adds a route to the group with the group's prefix prepended.
// Middleware is applied in order: router global middleware, group middleware, then route-specific middleware.
func (g *routerGroup) AddRoute(method HTTPMethod, pattern string, handler http.HandlerFunc, routeID RouteID, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := slices.Concat(g.middleware, middleware)
	fullPattern := g.prefix + normalizePath(pattern)
	if metadata == nil {
		metadata = &RouteMetadata{}
	}
	metadata.originalPattern = pattern
	g.router.AddRoute(method, fullPattern, handler, routeID, metadata, allMiddleware...)
}

func (r *router) applyMiddleware(mws []Middleware, h http.Handler) http.Handler {
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
