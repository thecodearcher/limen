import type { StrictConfig } from "../../define-plugin";
import { defineClientPlugin, schema } from "../../define-plugin";
import { sessionStore } from "../../session-store";
import { effect } from "../../stores";
import { resolveDefaultStorage } from "../bearer";
import { type AdminRoutes, buildBanRoutes, buildCoreRoutes, buildImpersonationRoutes } from "./routes";
import type { AdminPluginConfig, AdminSessionFields } from "./types";

const DEFAULT_IMPERSONATOR_STORAGE_KEY = "limen.admin.impersonator";

export function adminPlugin<
  const Config extends StrictConfig<Config, AdminPluginConfig> = Record<never, never>,
  TFields = unknown,
>(config?: Config) {
  const stash =
    config?.impersonation !== false
      ? (config?.storage ?? resolveDefaultStorage(config?.storageKey ?? DEFAULT_IMPERSONATOR_STORAGE_KEY))
      : null;

  const routes = [
    ...buildCoreRoutes<TFields>(),
    ...(config?.bans !== false ? buildBanRoutes<TFields>() : []),
    ...(stash !== null ? buildImpersonationRoutes<TFields>(stash) : []),
  ] as unknown as AdminRoutes<Config, TFields>;

  return defineClientPlugin({
    id: "admin",
    basePath: "/admin",
    routes,
    schema: schema<{
      user: { roles: string[]; permissions: string[] };
      session: AdminSessionFields<TFields>;
    }>(),
    effects:
      stash === null
        ? []
        : [
            effect(sessionStore, (session) => {
              if (session.data === null) {
                stash.clear();
              }
            }),
          ],
  });
}

export type {
  AdminPluginConfig,
  AdminRole,
  AdminSessionFields,
  AdminUser,
  AdminUserWithRoles,
  BanFields,
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
