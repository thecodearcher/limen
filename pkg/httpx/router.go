package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path"
	"strings"
)

type paramsContextKey struct{}

type HTTPMethod string

// HTTP method constants for method table
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
	// The path segment this node represents
	path string

	// Per-method handlers (7 methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
	// handlers [8]http.HandlerFunc

	// Per-method route metadata
	routes [8]*Route

	// Child nodes keyed by their path segments
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
	root *RadixNode
	// Exact fast path for static routes: map["METHOD PATH"]handler
	exactRoutes      map[string]http.HandlerFunc
	globalMiddleware []Middleware
}

// Route represents a single route with its handler and metadata
type Route struct {
	Method      HTTPMethod
	Pattern     string
	Handler     http.HandlerFunc
	RouteID     RouteID
	Description string
	Middleware  []Middleware
}

// RouteID is a unique identifier for each route
type RouteID string

// NewRouter creates a new radix tree router instance
func NewRouter(globalMiddleware ...Middleware) *Router {
	return &Router{
		root: &RadixNode{
			children: make(map[string]*RadixNode),
		},
		exactRoutes:      make(map[string]http.HandlerFunc),
		globalMiddleware: globalMiddleware,
	}
}

// AddRoute adds a new route to the radix tree
// Middleware is applied in order: global middleware first, then route-specific middleware
func (r *Router) AddRoute(method HTTPMethod, pattern string, handler http.HandlerFunc, routeID RouteID, middleware ...Middleware) {
	route := &Route{
		Method:  method,
		Pattern: pattern,
		Handler: handler,
		RouteID: routeID,
		// Description: description,
		Middleware: middleware,
	}

	// Apply middleware to create wrapped handler
	wrappedHandler := r.wrapHandler(handler, middleware)

	// Check if this is a static route (no parameters)
	if !strings.Contains(pattern, ":") && method != MethodANY {
		// Add to exact fast path with middleware applied
		key := string(method) + " " + pattern
		r.exactRoutes[key] = wrappedHandler
	}

	// Split path into segments, removing empty segments
	segments := r.splitPath(pattern)
	r.insertRoute(route, segments)
}

// insertRoute iteratively inserts a route into the radix tree
func (r *Router) insertRoute(route *Route, segments []string) {
	current := r.root

	// Iterate through each segment
	for _, segment := range segments {
		// Handle parameter segments
		if strings.HasPrefix(segment, ":") {
			current = r.handleParameterSegment(current, segment)
			continue
		}

		// Handle static segments
		current = r.handleStaticSegment(current, segment)
	}

	// Set the handler and route at the final node for the specific method
	methodIdx := methodIndex[route.Method]
	// current.handlers[methodIdx] = route.Handler
	current.routes[methodIdx] = route
}

// handleParameterSegment handles parameter segments with early returns
func (r *Router) handleParameterSegment(current *RadixNode, segment string) *RadixNode {
	paramName := segment[1:]

	// If we already have a parameter child, validate and use it
	if current.paramChild != nil {
		if current.paramChild.paramName != paramName {
			panic("conflicting parameter names at same path level")
		}
		return current.paramChild
	}

	// Create new parameter child
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
	// Look for existing child with this segment
	if child, exists := current.children[segment]; exists {
		return child
	}

	// Create new child node
	child := &RadixNode{
		path:     segment,
		children: make(map[string]*RadixNode),
	}
	current.children[segment] = child
	return child
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("ServeHTTP called\n %s %s\n", req.Method, req.URL.Path)
	fmt.Printf("Exact routes: %v\n", r.exactRoutes)
	// Try exact fast path first (O(1) for static routes)
	key := req.Method + " " + req.URL.Path
	if handler, exists := r.exactRoutes[key]; exists {
		handler(w, req)
		return
	}

	// Try HEAD -> GET fallback for exact routes
	if req.Method == "HEAD" {
		key = "GET " + req.URL.Path
		if handler, exists := r.exactRoutes[key]; exists {
			handler(w, req)
			return
		}
	}

	// Try deterministic pattern matching (no backtracking)
	segments := r.splitPath(req.URL.Path)
	route, params := r.matchRoute(segments, HTTPMethod(req.Method))
	if route != nil {
		r.handleRoute(w, req, route, params)
		return
	}

	// Try HEAD -> GET fallback for pattern routes
	if req.Method == "HEAD" {
		route, params = r.matchRoute(segments, MethodGET)
		if route != nil {
			r.handleRoute(w, req, route, params)
			return
		}
	}

	// 404 Not Found
	http.NotFound(w, req)
}

// Mount attaches a whole handler subtree under a fixed prefix using StripPrefix.
// No wildcard support needed; the mounted handler receives paths starting at its own root.
func (rt *Router) Mount(prefix string, h http.Handler, perMountMW []Middleware, hooks *Hooks) {
	prefix = NormalizeBasePath(prefix)

	h = applyMiddleware(append(rt.globalMiddleware, perMountMW...), h)
	stripped := http.StripPrefix(prefix, h)

	rt.AddRoute(MethodANY, prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hookCtx := &HookContext{
			Context:  r.Context(),
			Request:  r,
			Response: w,
			Method:   r.Method,
			Path:     r.URL.Path,
		}

		if hooks != nil && hooks.Before != nil {
			if !hooks.Before(hookCtx) {
				return
			}

			if hookCtx.bodyModified {
				bodyBytes, _ := json.Marshal(hookCtx.modifiedData)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				r = r.WithContext(context.WithValue(r.Context(), bodyContextKey{}, hookCtx.modifiedData))
			}
		}

		r, _ = parseAndStoreBody(r) // parse the body and store it in the request context

		fmt.Printf("Mounted handler called\n %s %s\n %s\n", r.Method, r.URL.Path, prefix)
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		rw := &responseWriter{
			ResponseWriter: w,
			wroteHeader:    false,
		}
		fmt.Printf("Stripped handler called\n %s %s\n %s\n", r.Method, r.URL.Path, prefix)
		stripped.ServeHTTP(rw, r)
		if hooks != nil && hooks.After != nil {
			hookCtx.StatusCode = rw.statusCode
			hooks.After(hookCtx)
		}
	}), "")
}

// wrapHandler applies global middleware and route-specific middleware to a handler
func (r *Router) wrapHandler(handler http.HandlerFunc, routeMiddleware []Middleware) http.HandlerFunc {
	// Combine global and route middleware (global first, then route-specific)
	allMiddleware := append(r.globalMiddleware, routeMiddleware...)
	wrapped := applyMiddleware(allMiddleware, http.HandlerFunc(handler))
	return wrapped.ServeHTTP
}

// handleRoute handles a matched route with parameters
func (r *Router) handleRoute(w http.ResponseWriter, req *http.Request, route *Route, params map[string]string) {
	// Add parameters to request context if any
	if len(params) > 0 {
		ctx := context.WithValue(req.Context(), paramsContextKey{}, params)
		req = req.WithContext(ctx)
	}

	// Apply middleware (global + route-specific) to the handler
	wrappedHandler := r.wrapHandler(route.Handler, route.Middleware)
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

	// Check if we have a handler for this method at the final node
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
	// Clean the path to handle // and ./.. shenanigans
	pathStr = path.Clean(pathStr)

	// Remove leading slash
	if pathStr == "/" || pathStr == "" {
		return []string{}
	}

	// Remove leading slash and split
	pathStr = strings.TrimPrefix(pathStr, "/")
	return strings.Split(pathStr, "/")
}

func applyMiddleware(mws []Middleware, h http.Handler) http.Handler {
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
