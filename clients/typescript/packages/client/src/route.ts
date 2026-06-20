import type { RouteContext } from "./context";
import type { LimenError } from "./errors";
import type { HTTPMethod } from "./types";

declare const INPUT: unique symbol;
declare const OUTPUT: unique symbol;

/**
 * Runs the route's default HTTP request and parser without session effects.
 * Pass an input override when a handler needs to omit client-only fields.
 */
export type HttpRunner<I> = <R = unknown>(input?: I) => Promise<R>;

/**
 * Custom route behavior for flows that need more than the default request
 * pipeline.
 */
export type RouteHandler<I, O, TFields = unknown> = (
  ctx: RouteContext<TFields>,
  input: I,
  http: HttpRunner<I>,
) => Promise<O>;

/**
 * Per-call options accepted as the final argument of every generated route
 * method.
 */
export type RouteCallOptions<O = unknown> = {
  /** Invoked with the resolved value after the call succeeds. */
  onSuccess?: (data: O) => void;
  /** Invoked with the error just before it is re-thrown. */
  onError?: (error: LimenError) => void;
};

/**
 * Declarative client route definition. The public client chain is derived from
 * `path` unless `as` is provided.
 */
export type RouteDef<I, O> = {
  method: HTTPMethod;
  path: `/${string}`;

  /** Dotted client chain override, e.g. `"twoFactor.getTotpUri"`. */
  as?: string;

  /** Merged under the caller's input before serialization. */
  defaults?: Partial<I>;
  /** SDK input → wire body/query. Defaults to shallow camelCase → snake_case. */
  serialize?: (input: I) => unknown;
  /** Raw response → typed output. Ignored when `parseSession` is set. */
  parse?: (raw: unknown) => O;
  /**
   * Parse the response as a session and store it when it contains a `user`.
   * Set `skipStore` to return the parsed session without writing it.
   */
  parseSession?: boolean;
  /** Resolve `path` from the client base path instead of the plugin base path. */
  absolute?: boolean;

  /** Input keys used as `:param` path values. */
  params?: readonly (keyof I & string)[];

  /** For `parseSession` routes, skip the session-store write. */
  skipStore?: boolean;
  /** Clear the session store on success. */
  clearSession?: boolean;
  /** Revalidate the session after success. */
  refetchSession?: boolean;

  /** Set `false` to keep the route out of the public client API. */
  expose?: boolean;

  /** Override the default route call behavior. */
  handler?: RouteHandler<I, O>;
};

/**
 * A route definition carrying its input and output types for inference.
 */
export type RouteDescriptor<I, O, D extends RouteDef<I, O> = RouteDef<I, O>> = D & {
  readonly [INPUT]: I;
  readonly [OUTPUT]: O;
};

/**
 * Loose route-descriptor constraint for route tuples and plugin definitions.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- accepts any route descriptor
export type AnyRouteDescriptor = RouteDescriptor<any, any, any>;

/**
 * Route definition with erased input/output types but typed fields.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- erases route I/O only
export type AnyRoute = RouteDef<any, any>;

export type InputOf<R> = R extends { readonly [INPUT]: infer I } ? I : never;
export type OutputOf<R> = R extends { readonly [OUTPUT]: infer O } ? O : never;

/**
 * Define an HTTP-backed client method for a plugin. The input/output types
 * become the generated method's argument and resolved value.
 *
 * @example
 *   route<VerifyInput, Session<TFields>>()({
 *     method: "POST",
 *     path: "/verify",
 *     parseSession: true,
 *   })
 */
export function route<I = void, O = unknown>() {
  return <const D extends RouteDef<I, O>>(def: D): RouteDescriptor<I, O, D> => def as RouteDescriptor<I, O, D>;
}
