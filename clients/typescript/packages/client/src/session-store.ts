import { atom, onMount, type ReadableAtom } from "nanostores";
import { LimenError } from "./errors";
import { createSessionSync } from "./session-sync";
import type { Session } from "./types";

export type SessionState<TFields = unknown> = {
  /** The current session, or `null` when signed out. */
  data: Session<TFields> | null;
  /** True while a `/me` fetch is in flight. */
  isPending: boolean;
  /**
   * The last non-401 failure (network error, 5xx, etc.). A 401 is not an error
   * — it resolves to `data: null`. `error` is cleared on the next successful or
   * 401 outcome.
   */
  error: LimenError | null;
};

type RefetchOptions = {
  /** Skip the fetch when the last hydration ran within this many milliseconds. */
  maxAgeMs?: number;
  /** Skip the fetch when the session is signed out. */
  skipSignedOut?: boolean;
};

export type SessionStore<TFields = unknown> = {
  readonly $session: ReadableAtom<SessionState<TFields>>;
  setData(session: Session<TFields> | null): void;
  /**
   * Re-validate the session from the server.
   */
  refetch(options?: RefetchOptions): Promise<void>;
};

type CreateSessionStoreArgs<TFields = unknown> = {
  hydrator: () => Promise<Session<TFields>>;
  initialSession?: Session<TFields> | null;
  /** Mirror session changes to other same-origin tabs. */
  crossTabSync?: boolean;
  /** Re-validate against `/me` when the tab returns to the foreground. */
  refetchOnWindowFocus?: boolean;
};

export function createSessionStore<TFields = unknown>(options: CreateSessionStoreArgs<TFields>): SessionStore<TFields> {
  const $session = atom<SessionState<TFields>>({
    data: options.initialSession ?? null,
    isPending: false,
    error: null,
  });

  let inFlightHydration: Promise<void> | null = null;
  // Bumped on every write so older async refresh results cannot overwrite newer state.
  let writeVersion = 0;
  // Timestamp the most recent session refetched.
  let lastRefreshedAt = 0;
  const isStale = (requestVersion: number): boolean => requestVersion !== writeVersion;

  const fetchSessionFromServer = async (): Promise<void> => {
    const requestVersion = ++writeVersion;
    $session.set({ data: $session.get().data, isPending: true, error: null });
    try {
      const session = await options.hydrator();

      if (isStale(requestVersion)) {
        return;
      }

      $session.set({ data: session, isPending: false, error: null });
    } catch (err) {
      if (isStale(requestVersion)) {
        return;
      }

      if (err instanceof LimenError && err.isUnauthorized) {
        // Not an error — the user is simply signed out.
        $session.set({ data: null, isPending: false, error: null });
        return;
      }

      const error =
        err instanceof LimenError
          ? err
          : new LimenError(err instanceof Error ? err.message : "Failed to load session", 0, "unknown");
      // Preserve the last known session; surface the failure via `error`.
      $session.set({ data: $session.get().data, isPending: false, error });
    }
  };

  const refetch = (options?: RefetchOptions): Promise<void> => {
    const { skipSignedOut, maxAgeMs } = options ?? {};
    if (skipSignedOut && $session.get().data === null) {
      return Promise.resolve();
    }

    if (maxAgeMs !== undefined && Date.now() - lastRefreshedAt < maxAgeMs) {
      return Promise.resolve();
    }

    if (!inFlightHydration) {
      inFlightHydration = fetchSessionFromServer().finally(() => {
        inFlightHydration = null;
        lastRefreshedAt = Date.now();
      });
    }
    return inFlightHydration;
  };

  const setData = (session: Session<TFields> | null): void => {
    writeVersion++; // supersede any in-flight hydrate so it can't overwrite this
    $session.set({ data: session, isPending: false, error: null });
  };

  const store: SessionStore<TFields> = { $session, setData, refetch };

  onMount($session, () =>
    createSessionSync(store, {
      fetchOnMount: options.initialSession === undefined,
      crossTabSync: options.crossTabSync ?? false,
      refetchOnWindowFocus: options.refetchOnWindowFocus ?? false,
    }),
  );

  return store;
}
