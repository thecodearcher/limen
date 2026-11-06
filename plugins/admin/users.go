package admin

import (
	"context"
	"slices"
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

func (p *adminPlugin) ListUsersWithRoles(ctx context.Context, actor *limen.User, query *ListUsersQuery, opts *limen.QueryOptions) (*limen.Page[*UserWithRoles], error) {
	page, err := p.ListUsers(ctx, actor, query, opts)
	if err != nil {
		return nil, err
	}

	users, err := p.attachUserRoles(ctx, page.Items)
	if err != nil {
		return nil, err
	}
	return &limen.Page[*UserWithRoles]{
		Items:      users,
		Total:      page.Total,
		Page:       page.Page,
		PerPage:    page.PerPage,
		TotalPages: page.TotalPages,
	}, nil
}

func (p *adminPlugin) GetUser(ctx context.Context, actor *limen.User, userID any) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:read")); err != nil {
		return nil, err
	}
	return p.core.DBAction.FindUserByID(ctx, userID)
}

func (p *adminPlugin) GetUserWithRoles(ctx context.Context, actor *limen.User, userID any) (*UserWithRoles, error) {
	user, err := p.GetUser(ctx, actor, userID)
	if err != nil {
		return nil, err
	}
	return p.withRoles(ctx, user)
}

func (p *adminPlugin) withRoles(ctx context.Context, user *limen.User) (*UserWithRoles, error) {
	roles, err := p.GetUserRoles(ctx, user)
	if err != nil {
		return nil, err
	}
	return &UserWithRoles{User: user, Roles: roles}, nil
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

func (p *adminPlugin) SetEmail(ctx context.Context, actor *limen.User, userID any, req *SetEmailRequest) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:set-email")); err != nil {
		return nil, err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := p.requireAdminTargetPermission(ctx, actor, user, "user:set-email-admins"); err != nil {
		return nil, err
	}

	email := limen.NormalizeEmail(req.Email)
	if email == user.Email {
		return user, nil
	}
	if req.SendVerification && !p.core.EmailVerificationEnabled() {
		return nil, ErrEmailVerificationDisabled
	}

	userSchema := p.core.Schema.User
	exists, err := p.core.Exists(ctx, userSchema, []limen.Where{limen.Eq(userSchema.GetEmailField(), email)})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	var verifiedAt *time.Time
	if req.EmailVerified && !req.SendVerification {
		now := time.Now()
		verifiedAt = &now
	}

	var updatedUser *limen.User
	var verification *limen.Verification
	err = p.core.WithTransaction(ctx, func(ctx context.Context) error {
		updatedUserModel, err := p.core.UpdateAndReturn(ctx, userSchema, map[limen.SchemaField]any{
			limen.UserSchemaEmailField:           email,
			limen.UserSchemaEmailVerifiedAtField: verifiedAt,
		}, []limen.Where{limen.Eq(userSchema.GetIDField(), user.ID)}, user.ID)
		if err != nil {
			return err
		}

		updatedUser = updatedUserModel.(*limen.User)
		if !req.SendVerification {
			return nil
		}
		verification, err = p.core.CreateEmailVerification(ctx, updatedUser)
		return err
	})
	if err != nil {
		return nil, err
	}

	if verification != nil {
		p.core.SendEmailVerificationMail(updatedUser, verification)
	}
	return updatedUser, nil
}

func (p *adminPlugin) BanUser(ctx context.Context, actor *limen.User, userID any, req *BanUserRequest) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:ban")); err != nil {
		return nil, err
	}
	if !p.config.bansEnabled {
		return nil, ErrBansNotEnabled
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.ID == actor.ID {
		return nil, ErrCannotBanSelf
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		return nil, ErrInvalidBanExpiry
	}

	model, err := p.core.UpdateAndReturn(ctx, p.userSchema, map[limen.SchemaField]any{
		UserSchemaBannedAtField:     time.Now(),
		UserSchemaBanReasonField:    req.Reason,
		UserSchemaBanExpiresAtField: req.ExpiresAt,
	}, []limen.Where{limen.Eq(p.userSchema.GetIDField(), user.ID)}, user.ID)
	if err != nil {
		return nil, err
	}

	if err := p.core.SessionManager.RevokeAllSessions(ctx, user.ID); err != nil {
		return nil, err
	}
	return model.(*limen.User), nil
}

func (p *adminPlugin) UnbanUser(ctx context.Context, actor *limen.User, userID any) (*limen.User, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:ban")); err != nil {
		return nil, err
	}
	if !p.config.bansEnabled {
		return nil, ErrBansNotEnabled
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	model, err := p.core.UpdateAndReturn(ctx, p.userSchema, map[limen.SchemaField]any{
		UserSchemaBannedAtField:     nil,
		UserSchemaBanReasonField:    nil,
		UserSchemaBanExpiresAtField: nil,
	}, []limen.Where{limen.Eq(p.userSchema.GetIDField(), user.ID)}, user.ID)
	if err != nil {
		return nil, err
	}
	return model.(*limen.User), nil
}

func (p *adminPlugin) SetUserPassword(ctx context.Context, actor *limen.User, userID any, req *SetUserPasswordRequest) error {
	if err := p.HasPermission(ctx, actor, access.P("user:set-password")); err != nil {
		return err
	}

	hasher, err := p.passwordHasher()
	if err != nil {
		return err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := p.requireAdminTargetPermission(ctx, actor, user, "user:set-password-admins"); err != nil {
		return err
	}

	if err := hasher.ValidatePassword(req.Password); err != nil {
		return err
	}
	hashedPassword, err := hasher.HashPassword(req.Password)
	if err != nil {
		return err
	}

	userSchema := p.core.Schema.User
	if err := p.core.Update(ctx, userSchema, map[limen.SchemaField]any{
		limen.UserSchemaPasswordField: hashedPassword,
	}, []limen.Where{limen.Eq(userSchema.GetIDField(), user.ID)}); err != nil {
		return err
	}
	if !req.RevokeSessions {
		return nil
	}
	return p.core.SessionManager.RevokeAllSessions(ctx, user.ID)
}

type passwordHasher interface {
	ValidatePassword(password string) error
	HashPassword(password string) (string, error)
}

func (p *adminPlugin) passwordHasher() (passwordHasher, error) {
	plugin, ok := p.core.GetPlugin(limen.PluginCredentialPassword)
	if !ok {
		return nil, ErrPasswordsNotEnabled
	}
	hasher, ok := plugin.(passwordHasher)
	if !ok {
		return nil, ErrPasswordsNotEnabled
	}
	return hasher, nil
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

	return p.core.DeleteUser(ctx, user.ID)
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

func (p *adminPlugin) serializeUser(user *limen.User) map[string]any {
	if p.config.userSerializer != nil {
		return p.config.userSerializer(user)
	}

	serialized := p.core.SerializeModel(p.core.Schema.User, user)
	if len(p.config.userFields) == 0 {
		return serialized
	}

	filtered := make(map[string]any, len(p.config.userFields))
	for _, field := range p.config.userFields {
		if value, ok := serialized[string(field)]; ok {
			filtered[string(field)] = value
		}
	}
	return filtered
}

func (p *adminPlugin) serializeUsersWithRoles(users []*UserWithRoles) []map[string]any {
	serialized := make([]map[string]any, 0, len(users))
	for _, user := range users {
		serialized = append(serialized, p.serializeUserWithRoles(user))
	}
	return serialized
}

func (p *adminPlugin) serializeUserWithRoles(user *UserWithRoles) map[string]any {
	names := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		names = append(names, role.Name())
	}
	slices.Sort(names)

	body := p.serializeUser(user.User)
	body["roles"] = names
	body["permissions"] = access.MergeRolePermissions(user.Roles...).Specs()
	return body
}
