package organization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/thecodearcher/limen"
)

func (o *organizationPlugin) CreateOrganization(ctx context.Context, user *limen.User, request *CreateOrganizationRequest) (*Organization, error) {
	request.Slug = o.applySlugNormalization(o.config.slugGenerator(request.Name, request.Slug))
	if request.Slug == "" {
		return nil, ErrInvalidSlug
	}

	existing, err := o.core.Exists(ctx, o.organizationSchema, []limen.Where{
		limen.Eq(o.organizationSchema.GetSlugField(), request.Slug),
	})

	if err != nil {
		return nil, err
	}

	if existing {
		return nil, ErrOrganizationSlugAlreadyExists
	}

	if o.config.allowOrgCreation != nil {
		if !o.config.allowOrgCreation(ctx, user) {
			return nil, ErrOrganizationCreationNotAllowed
		}
	}

	if err := o.hasUserReachedMaxOrganizations(ctx, user); err != nil {
		return nil, err
	}

	if o.hooks.BeforeCreateOrganization != nil {
		if err := o.hooks.BeforeCreateOrganization(ctx, user, request); err != nil {
			return nil, err
		}
	}

	var organization *Organization
	var owner *Member
	payload := &Organization{
		Name: request.Name,
		Slug: request.Slug,
		Logo: request.Logo,
	}
	if err := o.core.WithTransaction(ctx, func(ctx context.Context) error {
		organizationModel, err := o.core.CreateAndReturn(ctx, o.organizationSchema, payload, request.AdditionalFields, OrganizationSchemaSlugField)
		if err != nil {
			return err
		}
		organization = organizationModel.(*Organization)
		owner, err = o.createOrganizationOwner(ctx, user, organization)
		return err
	}); err != nil {
		return nil, err
	}

	if o.hooks.AfterCreateOrganization != nil {
		o.hooks.AfterCreateOrganization(ctx, organization, user, owner)
	}

	return organization, nil
}

func (o *organizationPlugin) createOrganizationOwner(ctx context.Context, user *limen.User, organization *Organization) (*Member, error) {
	memberModel, err := o.core.CreateAndReturn(ctx, o.memberSchema, &Member{
		OrganizationID: organization.ID,
		UserID:         user.ID,
	}, nil, MemberSchemaOrganizationIDField, MemberSchemaUserIDField)
	if err != nil {
		return nil, err
	}

	ownerRole := o.getOwnerRole()
	if ownerRole == nil {
		return nil, ErrOwnerRoleNotFound
	}

	ownerRoleName := ownerRole.Name()
	if err := o.core.Create(ctx, o.memberRoleSchema, &MemberRole{
		OrganizationID: organization.ID,
		MemberID:       memberModel.(*Member).ID,
		Role:           &ownerRoleName,
	}, nil); err != nil {
		return nil, err
	}

	return memberModel.(*Member), nil
}

func (o *organizationPlugin) hasUserReachedMaxOrganizations(ctx context.Context, user *limen.User) error {
	if o.config.maxOrgPerUser == 0 {
		return nil
	}

	count, err := o.core.Count(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetUserIDField(), user.ID),
	})

	if err != nil {
		return err
	}

	if count >= int64(o.config.maxOrgPerUser) {
		return ErrUserHasReachedMaxOrganizations
	}
	return nil
}

func (o *organizationPlugin) ListOrganizations(ctx context.Context, user *limen.User, filter *ListOrganizationsFilter, opts *limen.QueryOptions) (*limen.Page[*Organization], error) {
	memberships, err := o.core.FindMany(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetUserIDField(), user.ID),
	})

	if err != nil {
		return nil, err
	}

	if len(memberships) == 0 {
		return limen.EmptyPage[*Organization](opts), nil
	}

	orgIds := make([]any, len(memberships))
	for i, membership := range memberships {
		orgIds[i] = membership.(*Member).OrganizationID
	}

	conditions := []limen.Where{
		limen.In(o.organizationSchema.GetIDField(), orgIds),
	}

	if filter.Name != nil {
		conditions = append(conditions, limen.Contains(o.organizationSchema.GetNameField(), *filter.Name))
	}

	organizations, err := o.core.FindWithOptions(ctx, o.organizationSchema, conditions, opts)
	return limen.MapPage[*Organization](organizations), err
}

func (o *organizationPlugin) UpdateOrganization(ctx context.Context, user *limen.User, organizationID any, request *UpdateOrganizationRequest) (*Organization, error) {
	organization, err := o.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	if err := o.HasPermission(ctx, user, organization.ID, perms("organization:update")); err != nil {
		return nil, err
	}

	if request.Slug != nil {
		slug := o.applySlugNormalization(*request.Slug)
		if slug == "" {
			return nil, ErrInvalidSlug
		}

		taken, err := o.core.Exists(ctx, o.organizationSchema, []limen.Where{
			limen.Eq(o.organizationSchema.GetSlugField(), slug),
			limen.Ne(o.organizationSchema.GetIDField(), organization.ID),
		})
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrOrganizationSlugAlreadyExists
		}
		request.Slug = &slug
	}

	if o.hooks.BeforeUpdateOrganization != nil {
		if err := o.hooks.BeforeUpdateOrganization(ctx, user, organization, request); err != nil {
			return nil, err
		}
	}

	payload := make(map[limen.SchemaField]any)
	if request.Name != nil {
		payload[OrganizationSchemaNameField] = *request.Name
	}
	if request.Slug != nil {
		payload[OrganizationSchemaSlugField] = *request.Slug
	}
	if request.Logo != nil {
		payload[OrganizationSchemaLogoField] = *request.Logo
	}
	if request.Metadata != nil {
		encoded, err := json.Marshal(request.Metadata)
		if err != nil {
			return nil, err
		}
		payload[OrganizationSchemaMetadataField] = string(encoded)
	}

	updated, err := o.core.UpdateAndReturn(ctx, o.organizationSchema, payload, []limen.Where{
		limen.Eq(o.organizationSchema.GetIDField(), organization.ID),
	}, organization.ID, limen.WithUpdateAdditionalFields(request.AdditionalFields))
	if err != nil {
		return nil, err
	}

	updatedOrganization := updated.(*Organization)
	if o.hooks.AfterUpdateOrganization != nil {
		o.hooks.AfterUpdateOrganization(ctx, user, updatedOrganization)
	}
	return updatedOrganization, nil
}

func (o *organizationPlugin) DeleteOrganization(ctx context.Context, user *limen.User, organizationID any) error {
	organization, err := o.GetOrganization(ctx, organizationID)
	if err != nil {
		return err
	}

	if err := o.HasPermission(ctx, user, organization.ID, perms("organization:delete")); err != nil {
		return err
	}

	if o.hooks.BeforeDeleteOrganization != nil {
		if err := o.hooks.BeforeDeleteOrganization(ctx, user, organization); err != nil {
			return err
		}
	}

	if err := o.core.WithTransaction(ctx, func(ctx context.Context) error {
		type childTable struct {
			schema         limen.Schema
			organizationID string
		}

		children := []childTable{
			{o.memberRoleSchema, o.memberRoleSchema.GetOrganizationIDField()},
			{o.memberSchema, o.memberSchema.GetOrganizationIDField()},
			{o.invitationSchema, o.invitationSchema.GetOrganizationIDField()},
		}
		if o.config.customRolesEnabled {
			children = append(children, childTable{o.organizationRoleSchema, o.organizationRoleSchema.GetOrganizationIDField()})
		}

		for _, child := range children {
			if err := o.core.Delete(ctx, child.schema, []limen.Where{
				limen.Eq(child.organizationID, organization.ID),
			}); err != nil {
				return err
			}
		}

		if err := o.clearActiveOrganizationFromSessions(ctx, organization.ID, nil); err != nil {
			return err
		}

		return o.core.Delete(ctx, o.organizationSchema, []limen.Where{
			limen.Eq(o.organizationSchema.GetIDField(), organization.ID),
		})
	}); err != nil {
		return err
	}

	if o.hooks.AfterDeleteOrganization != nil {
		o.hooks.AfterDeleteOrganization(ctx, user, organization)
	}
	return nil
}

func (o *organizationPlugin) CheckSlugAvailability(ctx context.Context, slug string) (bool, error) {
	slug = o.applySlugNormalization(slug)
	if slug == "" {
		return false, ErrInvalidSlug
	}

	existing, err := o.core.Exists(ctx, o.organizationSchema, []limen.Where{
		limen.Eq(o.organizationSchema.GetSlugField(), slug),
	})
	if err != nil {
		return false, err
	}
	return !existing, nil
}

func (o *organizationPlugin) applySlugNormalization(slug string) string {
	if o.config.normalizeSlugs {
		return normalizeSlug(slug)
	}
	return strings.TrimSpace(slug)
}

func normalizeSlug(value string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		isAllowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !isAllowed {
			pendingHyphen = b.Len() > 0
			continue
		}
		if pendingHyphen {
			b.WriteRune('-')
			pendingHyphen = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

func defaultSlugGenerator(name string, providedSlug string) string {
	if slug := strings.TrimSpace(providedSlug); slug != "" {
		return slug
	}
	if slug := normalizeSlug(name); slug != "" {
		return slug
	}
	return strings.ToLower(limen.GenerateRandomString(12, limen.CharSetAlphanumeric))
}

func (o *organizationPlugin) resolveActiveOrganization(ctx context.Context, session *limen.Session) (*Organization, error) {
	organizationID, err := o.GetActiveOrganizationID(ctx, session)
	if err != nil {
		return nil, err
	}
	if organizationID == nil {
		return nil, ErrNoActiveOrganization
	}

	organization, err := o.GetOrganization(ctx, organizationID)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrNoActiveOrganization
		}
		return nil, err
	}
	return organization, nil
}

func (o *organizationPlugin) GetActiveOrganization(ctx context.Context, session *limen.Session, user *limen.User) (*Organization, error) {
	organization, err := o.resolveActiveOrganization(ctx, session)
	if err != nil {
		return nil, err
	}

	if err := o.CheckMemberExistsInOrganization(ctx, organization.ID, user.ID); err != nil {
		if errors.Is(err, ErrMemberNotInOrganization) {
			return nil, ErrNoActiveOrganization
		}
		return nil, err
	}
	return organization, nil
}

func (o *organizationPlugin) GetOrganization(ctx context.Context, organizationID any) (*Organization, error) {
	organization, err := o.core.FindOne(ctx, o.organizationSchema, []limen.Where{
		limen.Eq(o.organizationSchema.GetIDField(), organizationID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return organization.(*Organization), nil
}
