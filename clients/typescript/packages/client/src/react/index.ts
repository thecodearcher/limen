import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { SessionState } from "../session-store";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";
import { useStore } from "./react-store";

/**
 * An {@link AuthClient} augmented with React hooks.
 */
export type ReactAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & {
    /**
     * Reactively read the session store. Re-renders the component whenever
     * `{ data, isPending, error }` changes.
     */
    useSession: () => SessionState<TFields>;
  }
>;

/**
 * Create a Limen auth client with React hooks attached.
 */
export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): ReactAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);
  const useSession = (): SessionState<TFields> => useStore(client.$session);

  return Object.assign(client, { useSession }) as ReactAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
export { useStore } from "./react-store";
