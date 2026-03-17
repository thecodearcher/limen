package limen

import (
	"net/http"
	"slices"
)

// RouteBuilder provides a clean API for plugins to register routes.
type RouteBuilder struct {
	group *RouterGroup
	core  *LimenHTTPCore
}

// isRouteDisabled checks if a route ID is in the disabled list
func (b *RouteBuilder) isRouteDisabled(routeID RouteID, path string) bool {
	if b.core.config == nil || len(b.core.config.disabledPaths) == 0 {
		return false
	}

	routeDisabled := slices.Contains(b.core.config.disabledPaths, string(routeID))
	pathDisabled := slices.Contains(b.core.config.disabledPaths, path)
	return routeDisabled || pathDisabled
}

// AddRoute adds a route to the router
func (b *RouteBuilder) AddRoute(method HTTPMethod, path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}

	b.group.AddRoute(method, path, handler, routeID, metadata, middleware...)
}

// POST registers a POST route
func (b *RouteBuilder) POST(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.AddRoute(MethodPOST, path, routeID, handler, nil, middleware...)
}

// GET registers a GET route
func (b *RouteBuilder) GET(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.AddRoute(MethodGET, path, routeID, handler, nil, middleware...)
}

// PUT registers a PUT route
func (b *RouteBuilder) PUT(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.AddRoute(MethodPUT, path, routeID, handler, nil, middleware...)
}

// DELETE registers a DELETE route
func (b *RouteBuilder) DELETE(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.AddRoute(MethodDELETE, path, routeID, handler, nil, middleware...)
}

// PATCH registers a PATCH route
func (b *RouteBuilder) PATCH(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.AddRoute(MethodPATCH, path, routeID, handler, nil, middleware...)
}

// ProtectedPOST registers a POST route with session requirement
func (b *RouteBuilder) ProtectedPOST(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.POST(path, routeID, handler, allMiddleware...)
}

// ProtectedGET registers a GET route with session requirement
func (b *RouteBuilder) ProtectedGET(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.GET(path, routeID, handler, allMiddleware...)
}

// ProtectedPUT registers a PUT route with session requirement
func (b *RouteBuilder) ProtectedPUT(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.PUT(path, routeID, handler, allMiddleware...)
}

// ProtectedDELETE registers a DELETE route with session requirement
func (b *RouteBuilder) ProtectedDELETE(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.DELETE(path, routeID, handler, allMiddleware...)
}

// ProtectedPATCH registers a PATCH route with session requirement
func (b *RouteBuilder) ProtectedPATCH(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.PATCH(path, routeID, handler, allMiddleware...)
}
