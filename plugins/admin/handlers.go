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

	routeBuilder.ProtectedGET("/users", "admin:users-list", h.ListUsers)
	routeBuilder.ProtectedGET("/users/:id", "admin:users-get", h.GetUser)
	routeBuilder.ProtectedPOST("/users", "admin:users-create", h.CreateUser)
	routeBuilder.ProtectedPATCH("/users/:id", "admin:users-update", h.UpdateUser)
	routeBuilder.ProtectedDELETE("/users/:id", "admin:users-delete", h.DeleteUser)
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

func (h *adminHandlers) validateUserIDParam(v *limen.Validator) {
	v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
		return limen.ValidateClientIDValue(h.plugin.core, h.plugin.core.Schema.User, value)
	})
}

func (h *adminHandlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := limen.BindAndValidate[ListUsersQuery](w, r, h.responder, func(v *limen.Validator) {
		v.Field("search").Optional().String()
		v.Field("filter").Optional().Object()
	})
	if query == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sortable := append([]limen.SchemaField{limen.SchemaCreatedAtField, limen.UserSchemaEmailField}, h.plugin.config.sortableUserFields...)
	opts := limen.ParsePagination(r, limen.WithSortableFields(h.plugin.core.Schema.User, sortable...))
	page, err := h.plugin.ListUsers(r.Context(), session.User, query, opts)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, limen.SerializePage(h.plugin.core, h.plugin.core.Schema.User, page))
}

func (h *adminHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
	if body := limen.ValidateRequest(w, r, h.responder, h.validateUserIDParam); body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	user, err := h.plugin.GetUser(r.Context(), session.User, limen.GetParam(r, "id"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.core.Schema.User, user))
}

func (h *adminHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[CreateUserRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("email").Required().String().Email()
		v.Field("first_name").Optional().String()
		v.Field("last_name").Optional().String()
		v.Field("email_verified").Optional().Boolean()
	})
	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	user, err := h.plugin.CreateUser(r.Context(), session.User, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusCreated, h.plugin.core.SerializeModel(h.plugin.core.Schema.User, user))
}

func (h *adminHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[UpdateUserRequest](w, r, h.responder, func(v *limen.Validator) {
		h.validateUserIDParam(v)
		v.Field("first_name").Optional().String()
		v.Field("last_name").Optional().String()
		v.Field("email_verified").Optional().Boolean()
	})
	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	user, err := h.plugin.UpdateUser(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.core.Schema.User, user))
}

func (h *adminHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if body := limen.ValidateRequest(w, r, h.responder, h.validateUserIDParam); body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.DeleteUser(r.Context(), session.User, limen.GetParam(r, "id")); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
