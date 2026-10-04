package admin

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type adminHandlers struct {
	plugin    *adminPlugin
	responder *limen.Responder
}

func (p *adminPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/admin",
	}
}

func (p *adminPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	h := &adminHandlers{plugin: p, responder: httpCore.Responder}

	routeBuilder.ProtectedGET("/roles", "admin:roles-list", h.ListRoles, h.requirePermission("user:set-role"))
}

func (h *adminHandlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles := make([]map[string]any, 0, len(h.plugin.config.roles))
	for _, role := range h.plugin.config.roles {
		roles = append(roles, map[string]any{
			"name":        role.Name(),
			"permissions": role.Permissions(),
		})
	}
	h.responder.JSON(w, r, http.StatusOK, roles)
}
