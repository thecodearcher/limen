package admin

import (
	"context"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *adminPlugin) ListUsers(ctx context.Context, actor *limen.User, query *ListUsersQuery, opts *limen.QueryOptions) (*limen.Page[*limen.User], error) {
	if err := p.HasPermission(ctx, actor, access.P("user:list")); err != nil {
		return nil, err
	}

	schema := p.core.Schema.User
	if query == nil {
		query = &ListUsersQuery{}
	}

	var conditions []limen.Where
	if query.Search != "" {
		columns := append([]string{
			schema.GetEmailField(),
			schema.GetField(limen.UserSchemaFirstNameField),
			schema.GetField(limen.UserSchemaLastNameField),
		}, p.config.searchColumns...)

		for _, column := range columns {
			if column == "" {
				continue
			}
			condition := limen.Contains(column, query.Search)
			if len(conditions) > 0 {
				condition = condition.Or()
			}
			conditions = append(conditions, condition)
		}
	}

	for name, value := range query.Filters {
		filter, ok := p.userFilters[name]
		if !ok {
			return nil, newInvalidFilterError(name, "is not a supported filter")
		}
		filterConditions, err := filter(value)
		if err != nil {
			return nil, newInvalidFilterError(name, err.Error())
		}
		conditions = append(conditions, filterConditions...)
	}

	page, err := p.core.FindWithOptions(ctx, schema, conditions, opts)
	if err != nil {
		return nil, err
	}
	return limen.MapPage[*limen.User](page), nil
}

func (p *adminPlugin) GetUser(ctx context.Context, actor *limen.User, userID any) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:read")); err != nil {
		return nil, err
	}
	return p.core.DBAction.FindUserByID(ctx, userID)
}

func (p *adminPlugin) CreateUser(ctx context.Context, actor *limen.User, req *CreateUserRequest) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:create")); err != nil {
		return nil, err
	}

	schema := p.core.Schema.User
	email := limen.NormalizeEmail(req.Email)
	exists, err := p.core.Exists(ctx, schema, []limen.Where{limen.Eq(schema.GetEmailField(), email)})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	payload := map[limen.SchemaField]any{
		limen.UserSchemaEmailField: email,
	}
	if req.EmailVerified {
		payload[limen.UserSchemaEmailVerifiedAtField] = time.Now()
	}
	p.addNameFields(payload, req.FirstName, req.LastName)

	model, err := p.core.CreateAndReturn(ctx, schema, payload, req.AdditionalFields, limen.UserSchemaEmailField)
	if err != nil {
		return nil, err
	}
	return model.(*limen.User), nil
}

func (p *adminPlugin) UpdateUser(ctx context.Context, actor *limen.User, userID any, req *UpdateUserRequest) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:update")); err != nil {
		return nil, err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	payload := map[limen.SchemaField]any{}
	if req.EmailVerified != nil {
		switch {
		case !*req.EmailVerified:
			payload[limen.UserSchemaEmailVerifiedAtField] = nil
		case user.EmailVerifiedAt == nil:
			payload[limen.UserSchemaEmailVerifiedAtField] = time.Now()
		}
	}
	p.addNameFields(payload, req.FirstName, req.LastName)

	schema := p.core.Schema.User
	model, err := p.core.UpdateAndReturn(ctx, schema, payload, []limen.Where{
		limen.Eq(schema.GetIDField(), user.ID),
	}, user.ID, limen.WithUpdateAdditionalFields(req.AdditionalFields))
	if err != nil {
		return nil, err
	}
	return model.(*limen.User), nil
}

func (p *adminPlugin) DeleteUser(ctx context.Context, actor *limen.User, userID any) error {
	if err := p.HasPermission(ctx, actor, access.P("user:delete")); err != nil {
		return err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.ID == actor.ID {
		return ErrCannotDeleteSelf
	}

	isLastAdmin, err := p.isLastAdmin(ctx, user.ID)
	if err != nil {
		return err
	}
	if isLastAdmin {
		return ErrLastAdmin
	}

	return p.core.DeleteUser(ctx, user.ID)
}

func (p *adminPlugin) isLastAdmin(ctx context.Context, userID any) (bool, error) {
	isAdmin, err := p.core.Exists(ctx, p.userRoleSchema, []limen.Where{
		limen.Eq(p.userRoleSchema.GetUserIDField(), userID),
		limen.Eq(p.userRoleSchema.GetRoleField(), p.config.adminRole),
	})
	if err != nil || !isAdmin {
		return false, err
	}

	admins, err := p.core.Count(ctx, p.userRoleSchema, []limen.Where{
		limen.Eq(p.userRoleSchema.GetRoleField(), p.config.adminRole),
	})
	if err != nil {
		return false, err
	}
	return admins <= 1, nil
}

func (p *adminPlugin) addNameFields(payload map[limen.SchemaField]any, firstName, lastName *string) {
	schema := p.core.Schema.User
	for field, value := range map[limen.SchemaField]*string{
		limen.UserSchemaFirstNameField: firstName,
		limen.UserSchemaLastNameField:  lastName,
	} {
		if value == nil || schema.GetField(field) == "" {
			continue
		}
		if *value == "" {
			payload[field] = nil
			continue
		}
		payload[field] = *value
	}
}
