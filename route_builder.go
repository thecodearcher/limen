package aegis

import (
	"net/http"
	"slices"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

// RouteBuilder provides a clean API for plugins to register routes.
type RouteBuilder struct {
	group *httpx.RouterGroup
	core  *AegisHTTPCore
}

// isRouteDisabled checks if a route ID is in the disabled list
func (b *RouteBuilder) isRouteDisabled(routeID httpx.RouteID, path string) bool {
	if b.core.config == nil || len(b.core.config.disabledPaths) == 0 {
		return false
	}

	routeDisabled := slices.Contains(b.core.config.disabledPaths, string(routeID))
	pathDisabled := slices.Contains(b.core.config.disabledPaths, path)
	return routeDisabled || pathDisabled
}

// AddRoute adds a route to the router
func (b *RouteBuilder) AddRoute(method httpx.HTTPMethod, path string, routeID httpx.RouteID, handler http.HandlerFunc, metadata *httpx.RouteMetadata, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}

	b.group.AddRoute(method, path, handler, routeID, metadata, middleware...)
}

// POST registers a POST route
func (b *RouteBuilder) POST(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	b.AddRoute(httpx.MethodPOST, path, routeID, handler, nil, middleware...)
}

// GET registers a GET route
func (b *RouteBuilder) GET(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	b.AddRoute(httpx.MethodGET, path, routeID, handler, nil, middleware...)
}

// PUT registers a PUT route
func (b *RouteBuilder) PUT(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	b.AddRoute(httpx.MethodPUT, path, routeID, handler, nil, middleware...)
}

// DELETE registers a DELETE route
func (b *RouteBuilder) DELETE(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	b.AddRoute(httpx.MethodDELETE, path, routeID, handler, nil, middleware...)
}

// PATCH registers a PATCH route
func (b *RouteBuilder) PATCH(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	b.AddRoute(httpx.MethodPATCH, path, routeID, handler, nil, middleware...)
}

// ProtectedPOST registers a POST route with session requirement
func (b *RouteBuilder) ProtectedPOST(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	allMiddleware := append([]httpx.Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.POST(path, routeID, handler, allMiddleware...)
}

// ProtectedGET registers a GET route with session requirement
func (b *RouteBuilder) ProtectedGET(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	allMiddleware := append([]httpx.Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.GET(path, routeID, handler, allMiddleware...)
}

// ProtectedPUT registers a PUT route with session requirement
func (b *RouteBuilder) ProtectedPUT(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	allMiddleware := append([]httpx.Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.PUT(path, routeID, handler, allMiddleware...)
}

// ProtectedDELETE registers a DELETE route with session requirement
func (b *RouteBuilder) ProtectedDELETE(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	allMiddleware := append([]httpx.Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.DELETE(path, routeID, handler, allMiddleware...)
}

// ProtectedPATCH registers a PATCH route with session requirement
func (b *RouteBuilder) ProtectedPATCH(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	allMiddleware := append([]httpx.Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.PATCH(path, routeID, handler, allMiddleware...)
}
