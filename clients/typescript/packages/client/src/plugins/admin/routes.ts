import { defineRoutes } from "../../define-plugin";
import { camelizeKeys } from "../../helpers";
import { route } from "../../route";
import type { Page, Session } from "../../types";
import type { BearerStorage } from "../bearer";
import { tokensStore } from "../bearer/token-store";
import type {
  AdminPluginConfig,
  AdminRole,
  AdminSessionFields,
  AdminUser,
  AdminUserWithRoles,
  BanUserInput,
  CreateUserInput,
  ListUsersInput,
  RevokeSessionInput,
  SetUserEmailInput,
  SetUserPasswordInput,
  SetUserRolesInput,
  UpdateUserInput,
  UserIdInput,
  UserSession,
} from "./types";

function parseUserSession<TFields>(raw: unknown): UserSession<TFields> {
  const session = camelizeKeys<UserSession<TFields>>(raw);
  if (session.impersonatedBy) {
    session.impersonatedBy = camelizeKeys<AdminUser<TFields>>(session.impersonatedBy);
  }
  return session;
}

export function buildCoreRoutes<TFields>() {
  return defineRoutes(
    route<void, AdminRole[]>()({
      method: "GET",
      path: "/roles",
      as: "admin.listRoles",
    }),
    route<ListUsersInput, Page<AdminUserWithRoles<TFields>>>()({
      method: "GET",
      path: "/users",
      as: "admin.listUsers",
    }),
    route<UserIdInput, AdminUserWithRoles<TFields>>()({
      method: "GET",
      path: "/users/:id",
      as: "admin.getUser",
      params: ["id"],
    }),
    route<CreateUserInput, AdminUser<TFields>>()({
      method: "POST",
      path: "/users",
      as: "admin.createUser",
    }),
    route<UpdateUserInput, AdminUser<TFields>>()({
      method: "PATCH",
      path: "/users/:id",
      as: "admin.updateUser",
      params: ["id"],
    }),
    route<UserIdInput, void>()({
      method: "DELETE",
      path: "/users/:id",
      as: "admin.deleteUser",
      params: ["id"],
    }),
    route<SetUserRolesInput, AdminUserWithRoles<TFields>>()({
      method: "PUT",
      path: "/users/:id/roles",
      as: "admin.setRoles",
      params: ["id"],
    }),
    route<SetUserEmailInput, AdminUser<TFields>>()({
      method: "PUT",
      path: "/users/:id/email",
      as: "admin.setEmail",
      params: ["id"],
    }),
    route<SetUserPasswordInput, void>()({
      method: "PUT",
      path: "/users/:id/password",
      as: "admin.setPassword",
      params: ["id"],
    }),
    route<UserIdInput, UserSession<TFields>[]>()({
      method: "GET",
      path: "/users/:id/sessions",
      as: "admin.listUserSessions",
      params: ["id"],
      parse: (raw) => (Array.isArray(raw) ? raw.map(parseUserSession<TFields>) : []),
    }),
    route<UserIdInput, void>()({
      method: "DELETE",
      path: "/users/:id/sessions",
      as: "admin.revokeUserSessions",
      params: ["id"],
    }),
    route<RevokeSessionInput, void>()({
      method: "DELETE",
      path: "/sessions/:token",
      as: "admin.revokeSession",
      params: ["token"],
    }),
  );
}

export function buildBanRoutes<TFields>() {
  return defineRoutes(
    route<BanUserInput, AdminUser<TFields>>()({
      method: "POST",
      path: "/users/:id/ban",
      as: "admin.banUser",
      params: ["id"],
    }),
    route<UserIdInput, AdminUser<TFields>>()({
      method: "POST",
      path: "/users/:id/unban",
      as: "admin.unbanUser",
      params: ["id"],
    }),
  );
}

export function buildImpersonationRoutes<TFields>(stash: BearerStorage) {
  return defineRoutes(
    route<UserIdInput, Session<TFields, AdminSessionFields<TFields>>>()({
      method: "POST",
      path: "/users/:id/impersonate",
      as: "admin.impersonate",
      params: ["id"],
      handler: async (ctx, _input, http) => {
        const adminTokens = ctx.stores.get(tokensStore)?.current().data ?? null;
        const session = ctx.parseSession(await http());
        if (adminTokens !== null) {
          stash.set(adminTokens);
        }
        if (session !== false) {
          ctx.setSession(session);
        }
        return session as Session<TFields, AdminSessionFields<TFields>>;
      },
    }),
    route<void, void>()({
      method: "POST",
      path: "/impersonation/stop",
      as: "admin.stopImpersonating",
      handler: async (ctx, _input, http) => {
        try {
          await http();
        } finally {
          const adminTokens = stash.get();
          if (adminTokens !== null) {
            ctx.stores.get(tokensStore)?.setData(adminTokens);
            stash.clear();
          }
          await ctx.refetchSession();
        }
      },
    }),
  );
}

type CoreRoutes<TFields> = ReturnType<typeof buildCoreRoutes<TFields>>;
type BanRoutes<TFields> = ReturnType<typeof buildBanRoutes<TFields>>;
type ImpersonationRoutes<TFields> = ReturnType<typeof buildImpersonationRoutes<TFields>>;

export type AdminRoutes<Config extends AdminPluginConfig, TFields> = readonly [
  ...CoreRoutes<TFields>,
  ...(Config["bans"] extends true ? BanRoutes<TFields> : []),
  ...(Config["impersonation"] extends true ? ImpersonationRoutes<TFields> : []),
];
