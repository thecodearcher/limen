import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { SessionState } from "../session-store";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";
import type { ReactiveValue } from "./vue-store";
import { useStore } from "./vue-store";

/**
 * An {@link AuthClient} augmented with Vue composables.
 */
export type VueAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & {
    /**
     * Reactively read the session store as a readonly ref. Updates whenever
     * `{ data, isPending, error }` changes.
     */
    useSession: () => ReactiveValue<SessionState<TFields>>;
  }
>;

/**
 * Create a Limen auth client with Vue composables attached.
 */
export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): VueAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);
  const useSession = (): ReactiveValue<SessionState<TFields>> => useStore(client.$session);

  return Object.assign(client, { useSession }) as VueAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
export { useStore } from "./vue-store";
export type { ReactiveValue } from "./vue-store";
