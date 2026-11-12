import type { SortablePaginationInput, User } from "../../types";
import type { BearerStorage } from "../bearer";

export type AdminPluginConfig = {
  /** Mount the ban routes. Enable it when the server runs with `WithBans(true)`. */
  bans?: boolean;
  /** Mount the impersonation routes. Enable it when the server runs with `WithImpersonation(true)`. */
  impersonation?: boolean;
  /**
   * Where the admin's bearer tokens are kept while impersonating. Defaults to
   * `localStorage` when available, otherwise an in-memory store.
   */
  storage?: BearerStorage;
  /** Key used by the default storage. Ignored when `storage` is supplied. Defaults to `"limen.admin.impersonator"`. */
  storageKey?: string;
};

export type BanFields = {
  bannedAt: string | null;
  banReason: string | null;
  banExpiresAt: string | null;
};

export type AdminUser<TFields = unknown> = User<TFields> & Partial<BanFields>;

export type AdminUserWithRoles<TFields = unknown> = AdminUser<TFields> & {
  roles: string[];
  permissions: string[];
};

export type AdminRole = {
  name: string;
  permissions: Record<string, string[]>;
};

export type UserSession<TFields = unknown> = {
  id: string;
  createdAt: string;
  expiresAt: string;
  lastAccess?: string;
  ipAddress?: string;
  userAgent?: string;
  impersonatedBy?: AdminUser<TFields>;
};

export type UserIdInput = {
  id: string | number;
};

export type ListUsersInput =
  | (SortablePaginationInput & {
      search?: string;
      /** Sent as `filter[name]=value`. Names are the server's filter names. */
      filter?: Record<string, string>;
    })
  | void;

export type CreateUserInput = {
  email: string;
  firstName?: string;
  lastName?: string;
  emailVerified?: boolean;
  additionalFields?: Record<string, unknown>;
};

export type UpdateUserInput = UserIdInput & {
  firstName?: string;
  lastName?: string;
  emailVerified?: boolean;
  additionalFields?: Record<string, unknown>;
};

export type SetUserRolesInput = UserIdInput & {
  /** `null` resets the user to the default roles. */
  roles: string[] | null;
};

export type SetUserEmailInput = UserIdInput & {
  email: string;
  /** Ignored when `sendVerification` is set. */
  emailVerified?: boolean;
  sendVerification?: boolean;
};

export type SetUserPasswordInput = UserIdInput & {
  password: string;
  revokeSessions?: boolean;
};

export type BanUserInput = UserIdInput & {
  reason?: string;
  /** RFC 3339 timestamp. Omit for a permanent ban. */
  expiresAt?: string;
};

export type RevokeSessionInput = {
  token: string;
};
