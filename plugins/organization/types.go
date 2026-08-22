package organization

import (
	"context"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type CreateOrganizationRequest struct {
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Logo             *string        `json:"logo,omitempty"`
	AdditionalFields map[string]any `json:"-"`
}

type ListOrganizationsFilter struct {
	Name *string `json:"name,omitempty"`
}

type UpdateOrganizationRequest struct {
	Name             *string        `json:"name,omitempty"`
	Slug             *string        `json:"slug,omitempty"`
	Logo             *string        `json:"logo,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	AdditionalFields map[string]any `json:"-"`
}

type SendInvitationMailData struct {
	Inviter      *limen.User
	Organization *Organization
	Invitation   *Invitation
}

type MaxMembersPerOrganizationFunc func(ctx context.Context, organization *Organization) int

type MaxRolesPerOrganizationFunc func(ctx context.Context, organization *Organization) int

type SlugGeneratorFunc func(name string, providedSlug string) string

type EmbeddedUserSerializerFunc func(user *limen.User) map[string]any

type EmbeddedOrganizationSerializerFunc func(organization *Organization) map[string]any

type CreateOrganizationRoleRequest struct {
	Name        string              `json:"name"`
	Description *string             `json:"description,omitempty"`
	Permissions map[string][]string `json:"permissions"`
}

type UpdateOrganizationRoleRequest struct {
	Description *string             `json:"description,omitempty"`
	Permissions map[string][]string `json:"permissions,omitempty"`
}

type config struct {
	accessControl                  *access.AccessControl
	roles                          []access.Role
	slugGenerator                  SlugGeneratorFunc
	normalizeSlugs                 bool
	hooks                          Hooks
	ownerRole                      string
	maxOrgPerUser                  int
	maxMembersPerOrganization      any
	allowOrgCreation               func(ctx context.Context, user *limen.User) bool
	sendInvitationMail             func(ctx context.Context, data *SendInvitationMailData)
	cancelPendingInviteOnNewInvite bool
	invitationExpirationSeconds    int
	customRolesEnabled             bool
	maxRolesPerOrganization        any
	embeddedUser                   any
	embeddedOrganization           any
}

type Hooks struct {
	BeforeCreateOrganization func(ctx context.Context, user *limen.User, request *CreateOrganizationRequest) error
	AfterCreateOrganization  func(ctx context.Context, organization *Organization, user *limen.User, owner *Member)

	BeforeUpdateOrganization func(ctx context.Context, user *limen.User, organization *Organization, request *UpdateOrganizationRequest) error
	AfterUpdateOrganization  func(ctx context.Context, user *limen.User, organization *Organization)

	BeforeDeleteOrganization func(ctx context.Context, user *limen.User, organization *Organization) error
	AfterDeleteOrganization  func(ctx context.Context, user *limen.User, organization *Organization)

	BeforeCreateInvitation func(ctx context.Context, user *limen.User, organization *Organization, request *CreateInvitationRequest) error
	AfterCreateInvitation  func(ctx context.Context, invitation *Invitation, user *limen.User, organization *Organization)

	BeforeCancelInvitation func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation) error
	AfterCancelInvitation  func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation)

	BeforeRespondToInvitation func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation, response InvitationResponse) error
	AfterRespondToInvitation  func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation, response InvitationResponse)

	BeforeAddMember func(ctx context.Context, organization *Organization, user *limen.User, role any) error
	AfterAddMember  func(ctx context.Context, organization *Organization, user *limen.User, member *Member)

	BeforeAssignMemberRole func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role) error
	AfterAssignMemberRole  func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role)

	BeforeRevokeMemberRole func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role) error
	AfterRevokeMemberRole  func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role)

	BeforeRemoveMember func(ctx context.Context, user *limen.User, organization *Organization, member *Member) error
	AfterRemoveMember  func(ctx context.Context, user *limen.User, organization *Organization, member *Member)

	BeforeCreateOrganizationRole func(ctx context.Context, user *limen.User, organization *Organization, request *CreateOrganizationRoleRequest) error
	AfterCreateOrganizationRole  func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole)

	BeforeUpdateOrganizationRole func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole, request *UpdateOrganizationRoleRequest) error
	AfterUpdateOrganizationRole  func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole)

	BeforeDeleteOrganizationRole func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole) error
	AfterDeleteOrganizationRole  func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole)
}

type ConfigOption func(*config)

// WithSlugGenerator derives the slug from the name and the client-provided slug,
// which is empty when none was sent.
func WithSlugGenerator(slugGenerator SlugGeneratorFunc) ConfigOption {
	return func(c *config) {
		c.slugGenerator = slugGenerator
	}
}

// WithSlugNormalization normalizes slugs before storage and lookup: lowercase,
// with runs of characters outside [a-z0-9] collapsed into single hyphens.
func WithSlugNormalization(enabled bool) ConfigOption {
	return func(c *config) {
		c.normalizeSlugs = enabled
	}
}

func WithHooks(hooks Hooks) ConfigOption {
	return func(c *config) {
		c.hooks = hooks
	}
}

func WithRoles(roles ...access.Role) ConfigOption {
	return func(c *config) {
		c.roles = roles
	}
}

func WithAccessControl(accessControl *access.AccessControl) ConfigOption {
	return func(c *config) {
		c.accessControl = accessControl
	}
}

func WithCreatorRole(ownerRole string) ConfigOption {
	return func(c *config) {
		c.ownerRole = ownerRole
	}
}

// WithMaxOrgPerUser sets the maximum number of organizations a user can be a member of.
// If set to 0, there is no limit.
func WithMaxOrgPerUser(maxOrgPerUser int) ConfigOption {
	return func(c *config) {
		c.maxOrgPerUser = maxOrgPerUser
	}
}

func WithAllowOrgCreation(allowOrgCreation func(ctx context.Context, user *limen.User) bool) ConfigOption {
	return func(c *config) {
		c.allowOrgCreation = allowOrgCreation
	}
}

func WithSendInvitationMail(sendInvitationMail func(ctx context.Context, data *SendInvitationMailData)) ConfigOption {
	return func(c *config) {
		c.sendInvitationMail = sendInvitationMail
	}
}

func WithCancelPendingInviteOnNewInvite(cancelPendingInviteOnNewInvite bool) ConfigOption {
	return func(c *config) {
		c.cancelPendingInviteOnNewInvite = cancelPendingInviteOnNewInvite
	}
}

func WithInvitationExpiration(invitationExpirationSeconds int) ConfigOption {
	return func(c *config) {
		c.invitationExpirationSeconds = invitationExpirationSeconds
	}
}

// WithMaxMembersPerOrganization sets the maximum number of members a organization can have.
// If set to 0, there is no limit.
func WithMaxMembersPerOrganization(maxMembersPerOrganization int) ConfigOption {
	return func(c *config) {
		c.maxMembersPerOrganization = maxMembersPerOrganization
	}
}

func WithMaxMembersPerOrganizationFunc(maxMembersPerOrganizationFunc MaxMembersPerOrganizationFunc) ConfigOption {
	return func(c *config) {
		c.maxMembersPerOrganization = maxMembersPerOrganizationFunc
	}
}

// WithCustomRoles enables organization-defined roles. When disabled the organization_roles
// table is not registered and the role management routes are not mounted.
func WithCustomRoles(enabled bool) ConfigOption {
	return func(c *config) {
		c.customRolesEnabled = enabled
	}
}

// WithMaxRolesPerOrganization sets the maximum number of custom roles an organization can define.
// If set to 0, there is no limit.
func WithMaxRolesPerOrganization(maxRolesPerOrganization int) ConfigOption {
	return func(c *config) {
		c.maxRolesPerOrganization = maxRolesPerOrganization
	}
}

// WithMaxRolesPerOrganizationFunc sets the limit per organization at request time.
// Returning 0 means no limit.
func WithMaxRolesPerOrganizationFunc(maxRolesPerOrganizationFunc MaxRolesPerOrganizationFunc) ConfigOption {
	return func(c *config) {
		c.maxRolesPerOrganization = maxRolesPerOrganizationFunc
	}
}

// WithEmbeddedUserFields limits the nested user object that organization responses carry,
// such as an invitation's "inviter" or a member's "user", to the given fields.
// Never exposes more than the core user serializer does.
func WithEmbeddedUserFields(fields ...limen.SchemaField) ConfigOption {
	return func(c *config) {
		c.embeddedUser = fields
	}
}

// WithEmbeddedUserSerializer builds the nested user object that organization responses carry,
// such as an invitation's "inviter" or a member's "user".
func WithEmbeddedUserSerializer(serializer EmbeddedUserSerializerFunc) ConfigOption {
	return func(c *config) {
		c.embeddedUser = serializer
	}
}

// WithEmbeddedOrganizationFields limits the nested organization object that responses carry,
// such as an invitation's "organization", to the given fields.
func WithEmbeddedOrganizationFields(fields ...limen.SchemaField) ConfigOption {
	return func(c *config) {
		c.embeddedOrganization = fields
	}
}

// WithEmbeddedOrganizationSerializer builds the nested organization object that responses
// carry, such as an invitation's "organization".
func WithEmbeddedOrganizationSerializer(serializer EmbeddedOrganizationSerializerFunc) ConfigOption {
	return func(c *config) {
		c.embeddedOrganization = serializer
	}
}
