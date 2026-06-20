import type { FetchInit } from "./plugin";
import type { SessionStore } from "./session-store";
import type { ParseSession, RedirectFn, Session } from "./types";

/**
 * Context passed to route handlers and plugin actions.
 */
export type RouteContext<TFields = unknown> = {
  /**
   * Fetch a path relative to the plugin base path. Pass
   * `init.absolute = true` to resolve from the client base path instead.
   */
  fetch: <T>(path: string, init?: FetchInit) => Promise<T>;
  /**
   * Navigate to an absolute URL. Returns whether navigation happened.
   */
  readonly redirect: RedirectFn;
  /**
   * Parse a session-bearing response.
   */
  readonly parseSession: ParseSession<TFields>;
  /**
   * Write a session into the reactive store, or `null` to clear it.
   */
  setSession: (session: Session<TFields> | null) => void;
  /** Revalidate the current session. */
  refetchSession: () => Promise<void>;
  /** Reactive session store. */
  readonly store: SessionStore<TFields>;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- TFields erased at boundaries
export type AnyRouteContext = RouteContext<any>;
