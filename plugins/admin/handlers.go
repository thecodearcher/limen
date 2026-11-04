package admin

import (
	"maps"
	"net/http"
	"slices"

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

	routeBuilder.ProtectedGET("/roles", "admin:roles-list", h.ListRoles)

	routeBuilder.ProtectedGET("/users", "admin:users-list", h.ListUsers)
	routeBuilder.ProtectedGET("/users/:id", "admin:users-get", h.GetUser)
	routeBuilder.ProtectedPOST("/users", "admin:users-create", h.CreateUser)
	routeBuilder.ProtectedPATCH("/users/:id", "admin:users-update", h.UpdateUser)
	routeBuilder.ProtectedDELETE("/users/:id", "admin:users-delete", h.DeleteUser)
	routeBuilder.ProtectedPUT("/users/:id/roles", "admin:users-set-roles", h.SetRoles)
	routeBuilder.ProtectedPUT("/users/:id/email", "admin:users-set-email", h.SetEmail)
	routeBuilder.ProtectedGET("/users/:id/sessions", "admin:users-sessions-list", h.ListUserSessions)
	routeBuilder.ProtectedDELETE("/users/:id/sessions", "admin:users-sessions-revoke-all", h.RevokeUserSessions)
	routeBuilder.ProtectedDELETE("/sessions/:token", "admin:sessions-revoke", h.RevokeSession)
}

func (h *adminHandlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	roles, err := h.plugin.ListRoles(r.Context(), session.User)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	out := make([]map[string]any, 0, len(roles))
	for _, name := range slices.Sorted(maps.Keys(roles)) {
		out = append(out, map[string]any{"name": name, "permissions": roles[name]})
	}
	h.responder.JSON(w, r, http.StatusOK, out)
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
	page, err := h.plugin.ListUsersWithRoles(r.Context(), session.User, query, opts)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, &limen.Page[map[string]any]{
		Items:      h.plugin.serializeUsersWithRoles(page.Items),
		Total:      page.Total,
		Page:       page.Page,
		PerPage:    page.PerPage,
		TotalPages: page.TotalPages,
	})
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

	user, err := h.plugin.GetUserWithRoles(r.Context(), session.User, limen.GetParam(r, "id"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.serializeUserWithRoles(user))
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

	h.responder.JSON(w, r, http.StatusCreated, h.plugin.serializeUser(user))
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

	h.responder.JSON(w, r, http.StatusOK, h.plugin.serializeUser(user))
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

type setRolesBody struct {
	Roles []string `json:"roles"`
}

func (h *adminHandlers) SetRoles(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[setRolesBody](w, r, h.responder, func(v *limen.Validator) {
		h.validateUserIDParam(v)
		v.Field("roles").Nullable().Array().String()
	})
	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	user, err := h.plugin.SetRoles(r.Context(), session.User, limen.GetParam(r, "id"), body.Roles)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.serializeUserWithRoles(user))
}

func (h *adminHandlers) SetEmail(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[SetEmailRequest](w, r, h.responder, func(v *limen.Validator) {
		h.validateUserIDParam(v)
		v.Field("email").Required().String().Email()
		v.Field("email_verified").Optional().Boolean()
		v.Field("send_verification").Optional().Boolean()
	})
	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	user, err := h.plugin.SetEmail(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.serializeUser(user))
}

func (h *adminHandlers) ListUserSessions(w http.ResponseWriter, r *http.Request) {
	if body := limen.ValidateRequest(w, r, h.responder, h.validateUserIDParam); body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessions, err := h.plugin.ListUserSessions(r.Context(), session.User, limen.GetParam(r, "id"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.serializeUserSessions(sessions))
}

func (h *adminHandlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("token").Required().String()
	})
	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.RevokeSession(r.Context(), session.User, limen.GetParam(r, "token")); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *adminHandlers) RevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	if body := limen.ValidateRequest(w, r, h.responder, h.validateUserIDParam); body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.RevokeUserSessions(r.Context(), session.User, limen.GetParam(r, "id")); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
