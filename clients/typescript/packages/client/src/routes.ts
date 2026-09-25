import { defineClientPlugin, defineRoutes } from "./define-plugin";
import type { InferPluginContribution } from "./infer";
import { route } from "./route";
import type { Session } from "./types";

export type VerifyEmailInput = {
  token: string;
};

export type RequestEmailVerificationInput = {
  email?: string;
};

export type ActiveSession = {
  id: string | number;
  token: string;
  userId: unknown;
  createdAt: string;
  expiresAt: string;
  lastAccess: string;
  metadata?: Record<string, unknown>;
};

/**
 * Core routes available on every client.
 */
export function coreClientPlugin<TFields = unknown>() {
  const routes = defineRoutes(
    route<void, ActiveSession[]>()({
      method: "GET",
      path: "/sessions",
    }),
    route<void, void>()({
      method: "POST",
      path: "/signout",
      clearSession: true,
    }),
    route<void, void>()({
      method: "POST",
      path: "/revoke-sessions",
      clearSession: true,
    }),
    route<VerifyEmailInput, string | Session<TFields>>()({
      method: "POST",
      path: "/verify-email",
      parseSession: true,
      refetchSession: true,
    }),
    route<RequestEmailVerificationInput | void, string>()({
      method: "POST",
      path: "/email-verifications",
      as: "requestEmailVerification",
    }),
  );

  return defineClientPlugin({
    id: "core",
    basePath: "/",
    routes,
    actions: (ctx) => ({
      /**
       * Revalidate session state with `GET /me`, update `$session`, and return the
       * resolved value (`null` when signed out).
       *
       * Prefer subscribing to `$session` for reactive UI state (`data`, `isPending`,
       * `error`). Use `getSession()` when you need an awaited server re-check, such
       * as route guards or SSR revalidation after `initialSession`.
       */
      getSession: async (): Promise<Session<TFields> | null> => {
        await ctx.refetchSession();
        const state = ctx.store.$state.get();
        if (state.error) {
          throw state.error;
        }
        return state.data as Session<TFields> | null;
      },
    }),
  });
}

export type CoreContribution<TFields = unknown> = InferPluginContribution<ReturnType<typeof coreClientPlugin<TFields>>>;
