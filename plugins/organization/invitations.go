package organization

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type CreateInvitationRequest struct {
	Email            string         `json:"email"`
	Role             any            `json:"role"`
	Resend           bool           `json:"resend"`
	AdditionalFields map[string]any `json:"-"`
}

type FindPendingInvitationOptions struct {
	InvitationID    any
	Email           string
	OrganizationID  any
	InvitationToken string
}

type InvitationRelations struct {
	Inviter      bool
	Organization bool
}

type ListInvitationsOptions struct {
	*limen.QueryOptions
	Statuses []InvitationStatus `json:"statuses"`
}

func (o *organizationPlugin) CreateInvitation(ctx context.Context, user *limen.User, organization *Organization, req *CreateInvitationRequest) (*Invitation, error) {
	actor, err := o.loadMemberAccess(ctx, organization.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if err := actor.requirePermissions(perms("invitation:create")); err != nil {
		return nil, err
	}

	req.Email = limen.NormalizeEmail(req.Email)

	invitation, err := o.FindPendingInvitation(ctx, &FindPendingInvitationOptions{
		Email:          req.Email,
		OrganizationID: organization.ID,
	})
	if err != nil && !errors.Is(err, limen.ErrRecordNotFound) {
		return nil, err
	}

	if err := o.checkUserWithEmailAlreadyInOrganization(ctx, organization.ID, req.Email); err != nil {
		return nil, err
	}

	role, err := o.validateAndResolveInvitationRole(ctx, actor, organization, req.Role)
	if err != nil {
		return nil, err
	}

	if err := o.checkOrganizationMemberLimit(ctx, organization); err != nil {
		return nil, err
	}

	if o.config.hooks.BeforeCreateInvitation != nil {
		if err := o.config.hooks.BeforeCreateInvitation(ctx, user, organization, req); err != nil {
			return nil, err
		}
	}

	invitation, err = o.processInvitationCreation(ctx, user, organization, req, role, invitation)
	if err != nil {
		return nil, err
	}

	if invitation != nil {
		if err := o.attachInvitationRelations(ctx, organization, []*Invitation{invitation}, InvitationRelations{}); err != nil {
			return nil, err
		}
	}

	if o.config.hooks.AfterCreateInvitation != nil {
		o.config.hooks.AfterCreateInvitation(ctx, invitation, user, organization)
	}

	if invitation != nil {
		o.sendInvitationMail(ctx, user, organization, invitation)
	}

	return invitation, nil
}

func (o *organizationPlugin) FindPendingInvitation(ctx context.Context, options *FindPendingInvitationOptions) (*Invitation, error) {
	conditions := []limen.Where{limen.Eq(o.invitationSchema.GetStatusField(), InvitationStatusPending),
		limen.IsNull(o.invitationSchema.GetExpiresAtField()),
		limen.Gt(o.invitationSchema.GetExpiresAtField(), time.Now()).Or(),
	}
	if options.OrganizationID != nil {
		conditions = append(conditions, limen.Eq(o.invitationSchema.GetOrganizationIDField(), options.OrganizationID))
	}
	if options.Email != "" {
		conditions = append(conditions, limen.Eq(o.invitationSchema.GetEmailField(), limen.NormalizeEmail(options.Email)))
	}
	if options.InvitationToken != "" {
		conditions = append(conditions, limen.Eq(o.invitationSchema.GetTokenField(), options.InvitationToken))
	}
	if options.InvitationID != nil {
		conditions = append(conditions, limen.Eq(o.invitationSchema.GetIDField(), options.InvitationID))
	}
	invitationModel, err := o.core.FindOne(ctx, o.invitationSchema, conditions, []limen.OrderBy{{
		Column:    o.invitationSchema.GetCreatedAtField(),
		Direction: limen.OrderByDesc,
	}})

	if err != nil {
		return nil, err
	}

	invitation := invitationModel.(*Invitation)
	if invitation.ExpiresAt != nil && invitation.ExpiresAt.Before(time.Now()) {
		return nil, limen.ErrRecordNotFound
	}
	return invitation, nil
}

func (o *organizationPlugin) GetInvitationByToken(ctx context.Context, user *limen.User, invitationToken string) (*Invitation, error) {
	invitation, err := o.FindPendingInvitation(ctx, &FindPendingInvitationOptions{InvitationToken: invitationToken})
	if err != nil {
		return nil, err
	}

	organization, err := o.GetOrganization(ctx, invitation.OrganizationID)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(invitation.Email, user.Email) {
		if err := o.HasPermission(ctx, user, organization.ID, perms("invitation:read")); err != nil {
			return nil, err
		}
	}

	if err := o.attachInvitationRelations(ctx, organization, []*Invitation{invitation}, InvitationRelations{
		Inviter:      true,
		Organization: true,
	}); err != nil {
		return nil, err
	}
	return invitation, nil
}

type respondToInvitationPrep struct {
	invitation   *Invitation
	organization *Organization
	status       InvitationStatus
	accepted     bool
}

func (o *organizationPlugin) RespondToInvitation(ctx context.Context, user *limen.User, invitationToken string, response InvitationResponse) (*Invitation, error) {
	prep, err := o.prepareRespondToInvitation(ctx, user, invitationToken, response)
	if err != nil {
		return nil, err
	}

	if err := o.invokeRespondToInvitationBeforeHooks(ctx, user, prep, response); err != nil {
		return nil, err
	}

	member, err := o.commitRespondToInvitation(ctx, user, prep)
	if err != nil {
		return nil, err
	}

	o.invokeRespondToInvitationAfterHooks(ctx, user, prep, response, member)

	if err := o.attachInvitationRelations(ctx, prep.organization, []*Invitation{prep.invitation}, InvitationRelations{}); err != nil {
		return nil, err
	}
	return prep.invitation, nil
}

func (o *organizationPlugin) prepareRespondToInvitation(ctx context.Context, user *limen.User, invitationToken string, response InvitationResponse) (*respondToInvitationPrep, error) {
	invitation, err := o.FindPendingInvitation(ctx, &FindPendingInvitationOptions{
		InvitationToken: invitationToken,
	})
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(invitation.Email, user.Email) {
		return nil, ErrInvitationEmailMismatch
	}

	organization, err := o.GetOrganization(ctx, invitation.OrganizationID)
	if err != nil {
		return nil, err
	}

	status, err := resolveResponseToStatus(response)
	if err != nil {
		return nil, err
	}

	accepted := status == InvitationStatusAccepted
	if accepted {
		if err := o.checkMemberCanJoin(ctx, organization, user); err != nil {
			return nil, err
		}
	}

	return &respondToInvitationPrep{
		invitation:   invitation,
		organization: organization,
		status:       status,
		accepted:     accepted,
	}, nil
}

func (o *organizationPlugin) invokeRespondToInvitationBeforeHooks(ctx context.Context, user *limen.User, prep *respondToInvitationPrep, response InvitationResponse) error {
	if o.config.hooks.BeforeRespondToInvitation != nil {
		if err := o.config.hooks.BeforeRespondToInvitation(ctx, user, prep.organization, prep.invitation, response); err != nil {
			return err
		}
	}

	if prep.accepted && o.hooks.BeforeAddMember != nil {
		return o.hooks.BeforeAddMember(ctx, prep.organization, user, prep.invitation.Roles[0])
	}
	return nil
}

func (o *organizationPlugin) invokeRespondToInvitationAfterHooks(ctx context.Context, user *limen.User, prep *respondToInvitationPrep, response InvitationResponse, member *Member) {
	if member != nil && o.hooks.AfterAddMember != nil {
		o.hooks.AfterAddMember(ctx, prep.organization, user, member)
	}
	if o.config.hooks.AfterRespondToInvitation != nil {
		o.config.hooks.AfterRespondToInvitation(ctx, user, prep.organization, prep.invitation, response)
	}
}

func (o *organizationPlugin) commitRespondToInvitation(ctx context.Context, user *limen.User, prep *respondToInvitationPrep) (*Member, error) {
	var member *Member
	err := o.core.WithTransaction(ctx, func(ctx context.Context) error {
		prep.invitation.Status = prep.status
		result, err := o.core.UpdateWithResult(ctx, o.invitationSchema, map[limen.SchemaField]any{InvitationSchemaStatusField: prep.status}, []limen.Where{
			limen.Eq(o.invitationSchema.GetIDField(), prep.invitation.ID),
			limen.Eq(o.invitationSchema.GetStatusField(), InvitationStatusPending),
		})
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return ErrInvalidInvitation
		}
		if prep.accepted {
			member, err = o.insertMemberWithRole(ctx, prep.organization, user, prep.invitation.Roles[0])
			return err
		}
		return nil
	})
	return member, err
}

func resolveResponseToStatus(response InvitationResponse) (InvitationStatus, error) {
	var status InvitationStatus
	switch response {
	case InvitationResponseAccept:
		status = InvitationStatusAccepted
	case InvitationResponseReject:
		status = InvitationStatusRejected
	default:
		return "", ErrInvalidInvitationResponse
	}
	return status, nil
}

func (o *organizationPlugin) CancelPendingInvitation(ctx context.Context, user *limen.User, organization *Organization, invitationID any) (*Invitation, error) {
	if err := o.HasPermission(ctx, user, organization.ID, perms("invitation:cancel")); err != nil {
		return nil, err
	}

	invitation, err := o.FindPendingInvitation(ctx, &FindPendingInvitationOptions{
		InvitationID: invitationID,
	})
	if err != nil {
		return nil, err
	}

	if invitation.OrganizationID != organization.ID {
		return nil, ErrInvalidInvitation
	}

	if o.config.hooks.BeforeCancelInvitation != nil {
		if err := o.config.hooks.BeforeCancelInvitation(ctx, user, organization, invitation); err != nil {
			return nil, err
		}
	}

	invitation, err = o.cancelPendingInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}

	if o.config.hooks.AfterCancelInvitation != nil {
		o.config.hooks.AfterCancelInvitation(ctx, user, organization, invitation)
	}

	if err := o.attachInvitationRelations(ctx, organization, []*Invitation{invitation}, InvitationRelations{}); err != nil {
		return nil, err
	}

	return invitation, nil
}

func (o *organizationPlugin) ListInvitations(ctx context.Context, user *limen.User, organization *Organization, options *ListInvitationsOptions) (*limen.Page[*Invitation], error) {
	if err := o.HasPermission(ctx, user, organization.ID, perms("invitation:read")); err != nil {
		return nil, err
	}

	conditions := []limen.Where{limen.Eq(o.invitationSchema.GetOrganizationIDField(), organization.ID)}

	if len(options.Statuses) > 0 {
		anyStatus := make([]any, len(options.Statuses))
		for i, status := range options.Statuses {
			anyStatus[i] = status
		}
		conditions = append(conditions, limen.In(o.invitationSchema.GetStatusField(), anyStatus))
	}

	page, err := o.core.FindWithOptions(ctx, o.invitationSchema, conditions, options.QueryOptions)
	if err != nil {
		return nil, err
	}
	return limen.MapPage[*Invitation](page), nil
}

func (o *organizationPlugin) ListInvitationsWithRelations(ctx context.Context, user *limen.User, organization *Organization, options *ListInvitationsOptions) (*limen.Page[*Invitation], error) {
	page, err := o.ListInvitations(ctx, user, organization, options)
	if err != nil {
		return nil, err
	}
	if err := o.attachInvitationRelations(ctx, organization, page.Items, InvitationRelations{}); err != nil {
		return nil, err
	}
	return page, nil
}

func (o *organizationPlugin) attachInvitationRelations(ctx context.Context, organization *Organization, invitations []*Invitation, relations InvitationRelations) error {
	if len(invitations) == 0 {
		return nil
	}

	if err := o.attachInvitationRoles(ctx, organization, invitations); err != nil {
		return err
	}

	if relations.Organization {
		for _, invitation := range invitations {
			invitation.Organization = organization
		}
	}

	if !relations.Inviter {
		return nil
	}
	return o.attachInvitationInviters(ctx, invitations)
}

func (o *organizationPlugin) attachInvitationRoles(ctx context.Context, organization *Organization, invitations []*Invitation) error {
	identifiers := o.distinctRoleIdentifiers(invitations)
	if len(identifiers) == 0 {
		return nil
	}

	resolvedRoles, err := o.resolveRoles(ctx, organization, identifiers)
	if err != nil {
		return err
	}

	rolesByID := make(map[any]*access.Role, len(resolvedRoles))
	for _, role := range resolvedRoles {
		rolesByID[roleKey(role)] = role
	}

	for _, invitation := range invitations {
		invitation.ResolvedRoles = o.rolesForInvitation(invitation, rolesByID)
	}
	return nil
}

func (o *organizationPlugin) distinctRoleIdentifiers(invitations []*Invitation) []any {
	identifiers := make([]any, 0)
	seen := make(map[any]struct{})
	for _, invitation := range invitations {
		for _, identifier := range invitation.Roles {
			normalized := o.core.Schema.NormalizeIDValue(identifier)
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			identifiers = append(identifiers, normalized)
		}
	}
	return identifiers
}

func (o *organizationPlugin) rolesForInvitation(invitation *Invitation, rolesByID map[any]*access.Role) []*access.Role {
	roles := make([]*access.Role, 0, len(invitation.Roles))
	for _, identifier := range invitation.Roles {
		if role := rolesByID[o.core.Schema.NormalizeIDValue(identifier)]; role != nil {
			roles = append(roles, role)
		}
	}
	return roles
}

func (o *organizationPlugin) attachInvitationInviters(ctx context.Context, invitations []*Invitation) error {
	inviterIDs := make([]any, 0, len(invitations))
	seen := make(map[any]struct{})
	for _, invitation := range invitations {
		if invitation.InviterID == nil {
			continue
		}
		if _, ok := seen[invitation.InviterID]; ok {
			continue
		}
		seen[invitation.InviterID] = struct{}{}
		inviterIDs = append(inviterIDs, invitation.InviterID)
	}

	if len(inviterIDs) == 0 {
		return nil
	}

	users, err := o.core.FindMany(ctx, o.core.Schema.User, []limen.Where{
		limen.In(o.core.Schema.User.GetIDField(), inviterIDs),
	})
	if err != nil {
		return err
	}

	usersByID := make(map[any]*limen.User, len(users))
	for _, user := range limen.MapToSliceOfType[*limen.User](users) {
		usersByID[user.ID] = user
	}

	for _, invitation := range invitations {
		invitation.Inviter = usersByID[invitation.InviterID]
	}
	return nil
}

func (o *organizationPlugin) processInvitationCreation(ctx context.Context, user *limen.User, organization *Organization, req *CreateInvitationRequest, role any, invitation *Invitation) (*Invitation, error) {
	if invitation == nil {
		return o.createNewInvitation(ctx, user, organization, req, role)
	}

	if req.Resend {
		return o.refreshInvitation(ctx, invitation)
	}

	if !o.config.cancelPendingInviteOnNewInvite {
		return nil, ErrInvitationAlreadyExists
	}

	var newInvitation *Invitation
	if err := o.core.WithTransaction(ctx, func(ctx context.Context) error {
		if _, err := o.cancelPendingInvitation(ctx, invitation); err != nil {
			return err
		}

		invitation, err := o.createNewInvitation(ctx, user, organization, req, role)
		if err != nil {
			return err
		}
		newInvitation = invitation
		return nil
	}); err != nil {
		return nil, err
	}
	return newInvitation, nil
}

func (o *organizationPlugin) cancelPendingInvitation(ctx context.Context, invitation *Invitation) (*Invitation, error) {
	result, err := o.core.UpdateWithResult(ctx, o.invitationSchema, map[limen.SchemaField]any{InvitationSchemaStatusField: InvitationStatusCanceled}, []limen.Where{
		limen.Eq(o.invitationSchema.GetIDField(), invitation.ID),
		limen.Eq(o.invitationSchema.GetStatusField(), InvitationStatusPending),
	})
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, ErrInvalidInvitation
	}

	invitation.Status = InvitationStatusCanceled
	return invitation, nil
}

func (o *organizationPlugin) createNewInvitation(ctx context.Context, user *limen.User, organization *Organization, req *CreateInvitationRequest, role any) (*Invitation, error) {
	payload := &Invitation{
		InviterID:      user.ID,
		OrganizationID: organization.ID,
		Email:          limen.NormalizeEmail(req.Email),
		Status:         InvitationStatusPending,
		Roles:          []any{role},
		Token:          limen.GenerateRandomString(32, limen.CharSetAlphanumeric),
	}

	if o.config.invitationExpirationSeconds > 0 {
		expiresAt := time.Now().Add(time.Duration(o.config.invitationExpirationSeconds) * time.Second)
		payload.ExpiresAt = &expiresAt
	}

	invitation, err := o.core.CreateAndReturn(ctx, o.invitationSchema, payload, req.AdditionalFields, InvitationSchemaTokenField)
	if err != nil {
		return nil, err
	}

	return invitation.(*Invitation), nil
}

func (o *organizationPlugin) validateAndResolveInvitationRole(ctx context.Context, actor memberAccess, organization *Organization, role any) (any, error) {
	resolvedRoles, err := o.resolveRoles(ctx, organization, []any{role})
	if err != nil {
		return nil, err
	}

	if len(resolvedRoles) == 0 {
		return nil, ErrFailedToResolveRoles
	}

	resolvedRole := resolvedRoles[0]
	result := resolvedRole.ID()
	if result == nil {
		result = resolvedRole.Name()
	}

	if err := o.ensureCanGrantRoles(actor, []*access.Role{resolvedRole}); err != nil {
		if errors.Is(err, ErrUserCannotManageOwnerRole) {
			return nil, ErrUserCannotInviteOwner
		}
		return nil, err
	}
	return result, nil
}

func (o *organizationPlugin) refreshInvitation(ctx context.Context, invitation *Invitation) (*Invitation, error) {
	updatedPayload := make(map[limen.SchemaField]any)
	if o.config.invitationExpirationSeconds > 0 {
		expiresAt := time.Now().Add(time.Duration(o.config.invitationExpirationSeconds) * time.Second)
		updatedPayload[InvitationSchemaExpiresAtField] = &expiresAt
		invitation.ExpiresAt = &expiresAt
	}

	if len(updatedPayload) == 0 {
		return invitation, nil
	}

	result, err := o.core.UpdateWithResult(ctx, o.invitationSchema, updatedPayload, []limen.Where{
		limen.Eq(o.invitationSchema.GetIDField(), invitation.ID),
		limen.Eq(o.invitationSchema.GetStatusField(), InvitationStatusPending),
	})

	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, ErrInvalidInvitation
	}

	return invitation, nil
}

func (o *organizationPlugin) sendInvitationMail(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation) {
	if o.config.sendInvitationMail == nil {
		return
	}

	o.config.sendInvitationMail(ctx, &SendInvitationMailData{
		Inviter:      user,
		Organization: organization,
		Invitation:   invitation,
	})
}

func (o *organizationPlugin) checkUserWithEmailAlreadyInOrganization(ctx context.Context, organizationID any, email string) error {
	user, err := o.core.FindOne(ctx, o.core.Schema.User, []limen.Where{
		limen.Eq(o.core.Schema.User.GetEmailField(), email),
	}, nil)
	if err != nil && !errors.Is(err, limen.ErrRecordNotFound) {
		return err
	}

	if user == nil || errors.Is(err, limen.ErrRecordNotFound) {
		return nil
	}

	err = o.CheckMemberExistsInOrganization(ctx, organizationID, user.(*limen.User).ID)
	if err != nil && !errors.Is(err, ErrMemberNotInOrganization) {
		return err
	}

	if errors.Is(err, ErrMemberNotInOrganization) {
		return nil
	}

	return ErrUserAlreadyInOrganization
}

func (o *organizationPlugin) checkOrganizationMemberLimit(ctx context.Context, organization *Organization) error {
	if o.config.maxMembersPerOrganization == 0 || o.config.maxMembersPerOrganization == nil {
		return nil
	}

	maxMembersPerOrganization := 0
	switch v := o.config.maxMembersPerOrganization.(type) {
	case int:
		maxMembersPerOrganization = v
	case MaxMembersPerOrganizationFunc:
		maxMembersPerOrganization = v(ctx, organization)
	}

	if maxMembersPerOrganization <= 0 {
		return nil
	}

	count, err := o.core.Count(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organization.ID),
	})
	if err != nil {
		return err
	}

	if count >= int64(maxMembersPerOrganization) {
		return ErrMaxMembersPerOrganizationReached
	}

	return nil
}
