import type { Readable } from "svelte/store";
import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { SessionState } from "../session-store";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions } from "../types";

/**
 * An {@link AuthClient} augmented with Svelte stores.
 */
export type SvelteAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & {
    /**
     * The reactive session as a Svelte readable store. Use it with `$`:
     * `$session.data`, `$session.isPending`, `$session.error`.
     */
    useSession: () => Readable<SessionState<TFields>>;
  }
>;

/**
 * Create a Limen auth client with Svelte stores attached.
 *
 * nanostores stores already satisfy Svelte's store contract (`subscribe(run)`
 * fires immediately and returns an unsubscribe), so `useSession` returns the
 * reactive `$session` store directly — no wrapper.
 */
export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): SvelteAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);
  const useSession = (): Readable<SessionState<TFields>> =>
    client.$session as unknown as Readable<SessionState<TFields>>;

  return Object.assign(client, { useSession }) as SvelteAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
