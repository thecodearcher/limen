import type { RouteContext } from "./context";
import type { InferRoutes, PathSegments } from "./infer";
import type { PluginHooks } from "./plugin";
import type { AnyRouteDescriptor, RouteDescriptor } from "./route";
import type { IsAny, UnionToIntersection } from "./type-utils";

/** Invoke one of the plugin's routes from `actions`. */
export type RunRoute = <I, O>(route: RouteDescriptor<I, O>, input: I) => Promise<O>;

/**
 * Client plugin contract
 */
export type ClientPlugin<
  Id extends string,
  BasePath extends string,
  Routes extends readonly AnyRouteDescriptor[],
  Actions,
> = {
  readonly id: Id;
  /** Default mount path relative to the client `basePath`, e.g. `"/magic-link"`; omit for the root (`""`). */
  readonly basePath?: BasePath;
  readonly routes: Routes;
  readonly hooks?: PluginHooks;
  readonly actions?: (ctx: RouteContext, run: RunRoute) => Actions;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- match any plugin shape
export type AnyClientPlugin = ClientPlugin<string, string, readonly AnyRouteDescriptor[], any>;

/**
 * Register a plugin's routes while preserving each route's input/output types
 * for the generated client API.
 */
export function defineRoutes<Routes extends readonly AnyRouteDescriptor[]>(...routes: Routes): Routes {
  return routes;
}

/**
 * Define a client plugin for `createAuthClient`.
 */
export function defineClientPlugin<
  const Id extends string,
  const Routes extends readonly AnyRouteDescriptor[],
  Actions = Record<never, never>,
  const BasePath extends string = "",
>(def: ClientPlugin<Id, BasePath, Routes, Actions>): ClientPlugin<Id, BasePath, Routes, Actions> {
  return {
    id: def.id,
    basePath: (def.basePath ?? "") as BasePath,
    routes: def.routes,
    ...(def.hooks !== undefined ? { hooks: def.hooks } : {}),
    ...(def.actions !== undefined ? { actions: def.actions } : {}),
  };
}

/**
 * The client-only methods a plugin contributes. Widened `any` contributes
 * nothing so the client does not become permissive.
 */
type ActionsOf<P> = P extends { actions?: (ctx: never, run: never) => infer A }
  ? IsAny<A> extends true
    ? unknown
    : A extends object
      ? A
      : unknown
  : unknown;

export type InferPluginContribution<P> =
  P extends ClientPlugin<infer _Id, infer BasePath, infer Routes, infer _Actions>
    ? InferRoutes<Routes, PathSegments<BasePath>> & ActionsOf<P>
    : unknown;

export type CombinedClientContributions<Plugins extends readonly unknown[]> = UnionToIntersection<
  { [K in keyof Plugins]: InferPluginContribution<Plugins[K]> }[number]
>;
