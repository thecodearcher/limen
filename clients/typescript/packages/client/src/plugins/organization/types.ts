import type { FieldsOf } from "../../define-plugin";
import type { PaginationInput, User } from "../../types";

export type OrganizationPermissions = Record<string, string[]>;

/**
 * The app's own columns on the plugin's models, declared with `fields()`.
 */
export type OrganizationModelFields = {
  organization?: object;
  member?: object;
  invitation?: object;
  role?: object;
};

export type Organization<F = unknown> = {
  id: string;
  name: string;
  slug: string;
  logo: string | null;
  metadata: Record<string, unknown> | null;
  createdAt: string;
  updatedAt: string;
} & FieldsOf<F, "organization">;

/**
 * Nested organization on a member or invitation. The fields the server embeds
 * are configurable, so every field beyond the identifier may be absent.
 */
export type EmbeddedOrganization<F = unknown> = Partial<Organization<F>>;

export type EmbeddedUser = Partial<Omit<User, "emailVerifiedAt">> & {
  firstName?: string;
  lastName?: string;
};

export type Member<F = unknown> = {
  id: string;
  roles?: string[];
  permissions?: string[];
  organization?: Organization<F>;
  user?: EmbeddedUser;
  createdAt: string;
  updatedAt: string;
} & FieldsOf<F, "member">;

export type InvitationStatus = "pending" | "accepted" | "rejected" | "canceled";

export type Invitation<F = unknown> = {
  id: string;
  email: string;
  status: InvitationStatus;
  roles?: string[];
  expiresAt: string | null;
  isExpired: boolean;
  organization?: EmbeddedOrganization<F>;
  inviter?: EmbeddedUser;
  createdAt: string;
  updatedAt: string;
} & FieldsOf<F, "invitation">;

export type OrganizationRole<F = unknown> = {
  id: string;
  name: string;
  description: string | null;
  permissions: OrganizationPermissions;
  createdAt: string;
  updatedAt: string;
} & FieldsOf<F, "role">;

export type OrganizationPluginConfig = {
  /**
   * Mount the organization-defined role routes. Enable it when the server runs
   * with `WithCustomRoles(true)`; the routes are not registered otherwise.
   */
  customRoles?: boolean;
  /**
   * The app's own columns on the organization models, declared with `fields()`.
   * Type-only — it is never read at runtime.
   */
  fields?: OrganizationModelFields;
};

export type CreateOrganizationInput = {
  name: string;
  slug?: string;
  logo?: string;
  /**
   * Extra columns for the new organization, sent at the body root. The server
   * reads them through the schema's additional-fields function.
   */
  additionalFields?: Record<string, unknown>;
};

export type ListOrganizationsInput = (PaginationInput & { name?: string }) | void;

export type UpdateOrganizationInput = {
  id: string;
  name?: string;
  slug?: string;
  logo?: string;
  metadata?: Record<string, unknown>;
  additionalFields?: Record<string, unknown>;
};

export type DeleteOrganizationInput = {
  id: string;
};

export type CheckSlugInput = {
  slug: string;
};

export type CheckSlugResult = {
  available: boolean;
  /** The slug as the server normalized it. */
  slug: string;
};

export type SwitchOrganizationInput = {
  /** The organization to activate, or `null` to clear the active organization. */
  id: string | null;
};

export type LeaveOrganizationInput = {
  id: string;
};

export type ListMembersInput = PaginationInput | void;

export type MemberRoleInput = {
  memberId: string;
  role: string;
};

export type RemoveMemberInput = {
  memberId: string;
};

export type CreateInvitationInput = {
  email: string;
  role: string;
  /** Extend the pending invitation for this email instead of rejecting it. */
  resend?: boolean;
  additionalFields?: Record<string, unknown>;
};

export type InvitationTokenInput = {
  token: string;
};

export type CancelInvitationInput = {
  invitationId: string;
};

export type ListInvitationsInput = (PaginationInput & { statuses?: InvitationStatus[] }) | void;

export type CreateOrganizationRoleInput = {
  name: string;
  description?: string;
  permissions: OrganizationPermissions;
};

export type UpdateOrganizationRoleInput = {
  roleId: string;
  description?: string;
  permissions?: OrganizationPermissions;
};

export type ListOrganizationRolesInput = PaginationInput | void;

export type DeleteOrganizationRoleInput = {
  roleId: string;
};
