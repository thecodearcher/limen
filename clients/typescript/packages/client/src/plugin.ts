import type { KebabToCamel } from "./type-utils";
import type { HTTPMethod } from "./types";

/**
 * Route filter for hooks. Matches the stable route path, so client overrides do
 * not change which hooks run.
 */
export type RouteMatcher = string | readonly string[] | ((ctx: { path: string }) => boolean);

/**
 * Mutable request context passed to `beforeRequest` hooks. Return the updated
 * context, or throw to abort the request.
 */
export type RequestContext = {
  method: HTTPMethod;
  /** Path relative to `baseURL` (already includes the configured `basePath`). */
  fullPath: string;
  /** Path relative to the client `basePath`. */
  path: string;
  /** Stable route identifier, unaffected by client overrides. */
  routePath: string;
  /** Full request URL. */
  url: string;
  headers: Headers;
  body: unknown;
};

/**
 * Mutable response context passed to `afterResponse` hooks. The body has
 * already been envelope-unwrapped.
 */
export type ResponseContext = Omit<RequestContext, "url"> & {
  /** Status code. */
  status: number;
  /** Whether the response is successful. */
  ok: boolean;
};

export type BeforeRequestHook = {
  /** Optional route filter. Omit to run for every request. */
  match?: RouteMatcher;
  run: (req: RequestContext) => RequestContext | Promise<RequestContext>;
};

export type AfterResponseHook = {
  match?: RouteMatcher;
  /** Run even for failed responses. */
  allowOnFailure?: boolean;
  run: (res: ResponseContext) => ResponseContext | Promise<ResponseContext>;
};

/**
 * Request and response hooks contributed by a plugin.
 */
export type PluginHooks = {
  beforeRequest?: BeforeRequestHook[];
  afterResponse?: AfterResponseHook[];
};

/** Options accepted by `ctx.fetch(path, init)`. */
export type FetchInit = {
  method?: HTTPMethod;
  /** JSON body. The fetcher stringifies. */
  body?: unknown;
  /** Appended as `?k=v` query string. */
  query?: Record<string, string>;
  /** Extra headers to merge with the defaults (`Content-Type`, `Accept`). */
  headers?: HeadersInit;
  /**
   * Resolve `path` from the client base path instead of the plugin base path.
   */
  absolute?: boolean;
};

export type PluginIdOf<P> = P extends { readonly id: infer Id extends string } ? Id : never;

export type PluginClientOverride = {
  /** Replace the plugin's default base path (relative to the client `basePath`). */
  basePath?: string;
};

/**
 * Per-plugin client overrides, keyed by camelCased plugin id.
 */
export type PluginOverrides<Plugins extends readonly unknown[]> = Partial<
  Record<KebabToCamel<PluginIdOf<Plugins[number]>>, PluginClientOverride>
>;
