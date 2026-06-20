import type { Accessor } from "solid-js";
import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { SessionState } from "../session-store";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";
import { useStore } from "./solid-store";

/**
 * An {@link AuthClient} augmented with Solid primitives.
 */
export type SolidAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & {
    /**
     * Reactively read the session store as a Solid accessor. Updates whenever
     * `{ data, isPending, error }` changes.
     */
    useSession: () => Accessor<SessionState<TFields>>;
  }
>;

/**
 * Create a Limen auth client with Solid primitives attached.
 */
export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): SolidAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);
  const useSession = (): Accessor<SessionState<TFields>> => useStore(client.$session);

  return Object.assign(client, { useSession }) as SolidAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
export { useStore } from "./solid-store";
