import type { AnyRouteContext } from "./context";
import type { DataStore } from "./data-store";
import { createStore } from "./data-store";
import { LimenError } from "./errors";
import type { FetchInit } from "./plugin";

export type CreateAuthStoreOptions<T> = {
  ctx: AnyRouteContext;
  path: string;
  parse?: (raw: unknown) => T;
  initial?: T | null;
  fetchOnMount?: boolean;
  /** Options forwarded to `ctx.fetch`. Defaults to `GET`. */
  fetch?: FetchInit;
};

/**
 * Session-gated remote store: skips when signed out, treats 401 as `null`,
 * and fetches on mount by default.
 */
export function createAuthStore<T>(options: CreateAuthStoreOptions<T>): DataStore<T | null> {
  const { ctx, path, parse, initial = null, fetchOnMount = true, fetch: init } = options;

  return createStore<T | null>({
    initial,
    fetchOnMount,
    loader: async () => {
      const session = ctx.store.current();
      if (session.settled && session.data === null) {
        return null;
      }

      try {
        const raw = await ctx.fetch<unknown>(path, { method: "GET", ...init });
        return parse === undefined ? (raw as T) : parse(raw);
      } catch (error) {
        if (error instanceof LimenError && error.is("unauthorized")) {
          return null;
        }
        throw error;
      }
    },
  });
}
