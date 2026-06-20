import type { InputOf, OutputOf, RouteCallOptions } from "./route";
import type { KebabToCamel, Split, UnionToIntersection } from "./type-utils";

type IsParam<S extends string> = S extends `:${string}` ? true : false;

/**
 * Turn a route path literal into its camelCased client-chain segments, dropping
 * leading slashes and `:param` segments. `"/otp/send"` -> `["otp", "send"]`.
 */
export type PathSegments<P extends string> = P extends `/${infer Rest}`
  ? PathSegments<Rest>
  : P extends `${infer Head}/${infer Tail}`
    ? Head extends ""
      ? PathSegments<Tail>
      : IsParam<Head> extends true
        ? PathSegments<Tail>
        : [KebabToCamel<Head>, ...PathSegments<Tail>]
    : P extends ""
      ? []
      : IsParam<P> extends true
        ? []
        : [KebabToCamel<P>];

type PathOf<R> = R extends { path: infer P extends string } ? P : never;

/**
 * The resolved chain for one route: an absolute `as` wins, otherwise it is the
 * plugin's base-path segments followed by the route-path segments.
 */
type ChainSegments<R, BasePrefix extends readonly string[]> = R extends { as: infer A extends string }
  ? Split<A, ".">
  : [...BasePrefix, ...PathSegments<PathOf<R>>];

type Nest<Segs extends readonly string[], Fn> = Segs extends readonly [
  infer Head extends string,
  ...infer Rest extends readonly string[],
]
  ? // Avoid emitting an index signature when route paths widen to `string`.
    string extends Head
    ? unknown
    : { [K in Head]: Nest<Rest, Fn> }
  : Fn;

/**
 * A no-input route (`I` is `void`) takes no input — only the optional, trailing
 * call options; everything else takes `input` followed by the call options.
 * Options are always the trailing argument so the runtime can place them
 * unambiguously (a lone argument is always the route input).
 */
type InferRouteFn<I, O> = [I] extends [void]
  ? (input?: void, opts?: RouteCallOptions<O>) => Promise<O>
  : (input: I, opts?: RouteCallOptions<O>) => Promise<O>;

type IsExposed<R> = R extends { expose: false } ? false : true;

type RouteFn<R> = InferRouteFn<InputOf<R>, OutputOf<R>>;

type InferOneRoute<R, BasePrefix extends readonly string[]> =
  IsExposed<R> extends false ? unknown : Nest<ChainSegments<R, BasePrefix>, RouteFn<R>>;

/**
 * Infer the public API tree for a list of route descriptors, given the owning
 * plugin's base-path segments. Each route mounts a callable at its resolved
 * chain; sibling keys merge via intersection, so `signin.credential` and
 * `signup.credential` coexist and both autocomplete.
 */
export type InferRoutes<Routes extends readonly unknown[], BasePrefix extends readonly string[]> = UnionToIntersection<
  { [K in keyof Routes]: InferOneRoute<Routes[K], BasePrefix> }[number]
>;
