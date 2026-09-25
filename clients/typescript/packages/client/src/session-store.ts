import type { DataStore, StoreState } from "./data-store";
import { createStore } from "./data-store";
import { LimenError } from "./errors";
import type { FetchInit } from "./plugin";
import { createSessionSync } from "./session-sync";
import { storeRef } from "./stores";
import type { ParseSession, Session } from "./types";

/**
 * `error` holds the last non-401 failure (network error, 5xx, etc.). A 401 is
 * not an error — it resolves to `data: null`.
 */
export type SessionState<TFields = unknown> = StoreState<Session<TFields> | null>;

export type SessionStore<TFields = unknown> = DataStore<Session<TFields> | null>;

export const sessionStore = storeRef<Session | null>("session");

type CreateSessionStoreArgs<TFields = unknown> = {
  fetch: <T>(path: string, init?: FetchInit) => Promise<T>;
  parseSession: ParseSession<TFields>;
  initialSession?: Session<TFields> | null;
  /** Mirror session changes to other same-origin tabs. */
  crossTabSync?: boolean;
  /** Re-validate against `/me` when the tab returns to the foreground. */
  refetchOnWindowFocus?: boolean;
};

export function createSessionStore<TFields = unknown>(options: CreateSessionStoreArgs<TFields>): SessionStore<TFields> {
  return createStore<Session<TFields> | null>({
    initial: options.initialSession ?? null,
    settled: options.initialSession !== undefined,
    fetchOnMount: options.initialSession === undefined,
    loader: async () => {
      try {
        const raw = await options.fetch<unknown>("/me", { method: "GET" });
        const session = options.parseSession(raw);
        return session === false ? null : session;
      } catch (error) {
        // Not an error — the user is simply signed out.
        if (error instanceof LimenError && error.is("unauthorized")) {
          return null;
        }
        throw error;
      }
    },
    onMount: (store) =>
      createSessionSync(store, {
        crossTabSync: options.crossTabSync ?? false,
        refetchOnWindowFocus: options.refetchOnWindowFocus ?? false,
      }),
  });
}
