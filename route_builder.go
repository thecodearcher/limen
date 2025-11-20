package aegis

import (
	"net/http"
	"slices"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

// RouteBuilder provides a clean API for plugins to register routes.
type RouteBuilder struct {
	group *httpx.RouterGroup
	*AegisHTTPCore
}

// isRouteDisabled checks if a route ID is in the disabled list
func (b *RouteBuilder) isRouteDisabled(routeID httpx.RouteID, path string) bool {
	if b.Config == nil || len(b.Config.disabledPaths) == 0 {
		return false
	}

	routeDisabled := slices.Contains(b.Config.disabledPaths, string(routeID))
	pathDisabled := slices.Contains(b.Config.disabledPaths, path)
	return routeDisabled || pathDisabled
}

// POST registers a POST route
func (b *RouteBuilder) POST(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	b.group.AddRoute(httpx.MethodPOST, path, handler, routeID, middleware...)
}

// GET registers a GET route
func (b *RouteBuilder) GET(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	b.group.AddRoute(httpx.MethodGET, path, handler, routeID, middleware...)
}

// PUT registers a PUT route
func (b *RouteBuilder) PUT(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	b.group.AddRoute(httpx.MethodPUT, path, handler, routeID, middleware...)
}

// DELETE registers a DELETE route
func (b *RouteBuilder) DELETE(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	b.group.AddRoute(httpx.MethodDELETE, path, handler, routeID, middleware...)
}

// PATCH registers a PATCH route
func (b *RouteBuilder) PATCH(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	b.group.AddRoute(httpx.MethodPATCH, path, handler, routeID, middleware...)
}

// ProtectedPOST registers a POST route with session requirement
func (b *RouteBuilder) ProtectedPOST(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	allMiddleware := append([]httpx.Middleware{b.MiddlewareRequireSession()}, middleware...)
	b.POST(path, routeID, handler, allMiddleware...)
}

// ProtectedGET registers a GET route with session requirement
func (b *RouteBuilder) ProtectedGET(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	allMiddleware := append([]httpx.Middleware{b.MiddlewareRequireSession()}, middleware...)
	b.GET(path, routeID, handler, allMiddleware...)
}

// ProtectedPUT registers a PUT route with session requirement
func (b *RouteBuilder) ProtectedPUT(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	allMiddleware := append([]httpx.Middleware{b.MiddlewareRequireSession()}, middleware...)
	b.PUT(path, routeID, handler, allMiddleware...)
}

// ProtectedDELETE registers a DELETE route with session requirement
func (b *RouteBuilder) ProtectedDELETE(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	allMiddleware := append([]httpx.Middleware{b.MiddlewareRequireSession()}, middleware...)
	b.DELETE(path, routeID, handler, allMiddleware...)
}

// ProtectedPATCH registers a PATCH route with session requirement
func (b *RouteBuilder) ProtectedPATCH(path string, routeID httpx.RouteID, handler http.HandlerFunc, middleware ...httpx.Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}
	allMiddleware := append([]httpx.Middleware{b.MiddlewareRequireSession()}, middleware...)
	b.PATCH(path, routeID, handler, allMiddleware...)
}
