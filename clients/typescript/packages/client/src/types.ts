import type { ReadableAtom } from "nanostores";
import type { AnyClientPlugin, CombinedClientContributions } from "./define-plugin";
import type { FetcherFetchOptions } from "./fetcher";
import type { PluginOverrides } from "./plugin";
import type { CoreContribution } from "./routes";
import type { SessionState } from "./session-store";
import type { Prettify } from "./type-utils";

export type ClientFetchOptions = Partial<FetcherFetchOptions>;

export type CreateAuthClientOptions<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = {
  /** Server origin, e.g. `"http://localhost:8080"`. Trailing slash is stripped. */
  baseURL: string;
  /** Path where the Limen handler is mounted. Defaults to `"/auth"`. */
  basePath?: string;
  /** Options that modify how the SDK performs HTTP requests. */
  fetchOptions?: ClientFetchOptions;
  /**
   * Optional transformer for non-default session payloads.
   *
   * Provide this when your server returns custom user/session fields. It must
   * map the raw response into `Session<TFields>`.
   */
  parseSession?: ParseSession<TFields>;
  /**
   * How the SDK navigates the browser when a flow hands control to an external
   * page (e.g. an OAuth provider's authorization URL). Defaults to
   * `window.location.href = url` when a `window` is available; a no-op in
   * non-browser environments. Provide a custom function to integrate with a
   * client-side router.
   */
  redirectFn?: RedirectFn;
  /** Plugins to register. */
  plugins?: Plugins;
  /**
   * Response-envelope config. Set this when the server wraps successful or all
   * responses.
   */
  envelope?: EnvelopeConfig;
  /**
   * Per-plugin route overrides. Keys are camelCased plugin ids, e.g.
   * `{ magicLink: { basePath: "/passwordless" } }`.
   */
  overrides?: PluginOverrides<Plugins>;
  /**
   * SSR seed for the session store. Provide the session resolved server-side to
   * avoid a hydration flash. When provided, lazy hydration is skipped until you
   * call `getSession()` or the store revalidates.
   */
  initialSession?: Session<TFields> | null;
  /**
   * Keep session state in sync across browser tabs.
   * Enabled by default in browsers. Set `false` to disable.
   */
  crossTabSync?: boolean;
  /**
   * Refresh session state when the tab becomes active again.
   * Enabled by default in browsers. Set `false` to disable.
   */
  refetchOnWindowFocus?: boolean;
};

export type AuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  {
    readonly baseURL: string;
    readonly basePath: string;
    /**
     * Reactive session store holding `{ data, isPending, error }`. Read it with
     * `.get()` / `.listen()` or a framework `useStore`.
     */
    readonly $session: ReadableAtom<SessionState<TFields>>;
  } & CoreContribution<TFields> &
    CombinedClientContributions<Plugins>
>;

/**
 * The user object returned from `/me` and any session-bearing response.
 *
 * `TFields` lets consumers extend the shape with custom user fields. The
 * fields listed here are always present.
 *
 * @example
 *   type AppUser = User<{ firstName: string; orgId: string }>;
 */
export type User<TFields = unknown> = {
  id: string;
  email: string;
  emailVerifiedAt: string | null;
} & TFields;

export type Session<TFields = unknown> = {
  user: User<TFields>;
};

export type EnvelopeMode = "off" | "wrap-success" | "always";

export type EnvelopeFields = {
  data: string;
  message: string;
};

export type EnvelopeConfig = {
  mode: EnvelopeMode;
  fields?: EnvelopeFields;
};

export type ParseSession<TFields = unknown> = (raw: unknown) => Session<TFields>;

export type HTTPMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "HEAD" | "OPTIONS";

export type RedirectFn = (url: string) => boolean;
