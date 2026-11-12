import type { RouteContext } from "./context";
import type { PluginHooks } from "./plugin";
import type { AnyRouteDescriptor, RouteDescriptor } from "./route";
import type { PluginStores, StoreEffect } from "./stores";
import type { IsAny } from "./type-utils";

/** Invoke one of the plugin's routes from `actions`. */
export type RunRoute = <I, O>(route: RouteDescriptor<I, O>, input: I) => Promise<O>;

/**
 * The models a client plugin may contribute extra fields to. Each optional key
 * maps a model name to the extra fields the plugin adds.
 */
export type PluginSchema = {
  user?: object;
  session?: object;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- keeps RouteContext<any> assignment-compatible under AnyClientPlugin
type AnyUserFields = any;

/** The fields declared for one model, or `Fallback` when there are none. */
export type FieldsOf<F, M extends string, Fallback = unknown> =
  F extends Record<M, infer T> ? (T extends object ? T : Fallback) : Fallback;

/**
 * The `user` fields a plugin itself declared via `schema` — used to type the
 * plugin's own `ctx`, so an action can read its own contributions off
 * `ctx.store.$session` without a cast. A plugin without a `user` schema sees the
 * base context (`unknown`); widened `any` stays `any` so a concrete plugin
 * remains assignable to `AnyClientPlugin`.
 */
type OwnUserFields<Schema> = IsAny<Schema> extends true ? AnyUserFields : FieldsOf<Schema, "user">;

export type ClientPlugin<
  Id extends string,
  BasePath extends string,
  Routes extends readonly AnyRouteDescriptor[],
  Actions,
  Schema = unknown,
  Stores extends PluginStores = PluginStores,
> = {
  readonly id: Id;
  /** Default mount path relative to the client `basePath`, e.g. `"/magic-link"`; omit for the root (`""`). */
  readonly basePath?: BasePath;
  readonly routes: Routes;
  readonly hooks?: PluginHooks;
  readonly actions?: (ctx: RouteContext<OwnUserFields<Schema>>, run: RunRoute) => Actions;
  /**
   * Type-only declaration of the extra model fields this plugin contributes to
   * the client read surfaces — folded in only when the plugin is registered.
   * Declare it with {@link schema}.
   */
  readonly schema?: Schema;
  /**
   * Reactive stores this plugin owns, declared with {@link defineStores}. Names
   * are global to the client: two plugins cannot claim the same one.
   */
  readonly stores?: (ctx: RouteContext<OwnUserFields<Schema>>) => Stores;
  /** Reactions to store writes, declared with {@link effect}. */
  readonly effects?: readonly StoreEffect[];
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- match any plugin shape
export type AnyClientPlugin = ClientPlugin<string, string, readonly AnyRouteDescriptor[], any, any, any>;

export function defineRoutes<Routes extends readonly AnyRouteDescriptor[]>(...routes: Routes): Routes {
  return routes;
}

/** Register a plugin's stores, preserving each name for `$name` and `useName()`. */
export function defineStores<const Stores extends PluginStores>(stores: Stores): Stores {
  return stores;
}

export function defineClientPlugin<
  const Id extends string,
  const Routes extends readonly AnyRouteDescriptor[],
  Actions = Record<never, never>,
  const BasePath extends string = "",
  Schema = unknown,
  const Stores extends PluginStores = Record<never, never>,
>(
  def: ClientPlugin<Id, BasePath, Routes, Actions, Schema, Stores>,
): ClientPlugin<Id, BasePath, Routes, Actions, Schema, Stores> {
  return {
    id: def.id,
    basePath: (def.basePath ?? "") as BasePath,
    routes: def.routes,
    ...(def.hooks !== undefined ? { hooks: def.hooks } : {}),
    ...(def.actions !== undefined ? { actions: def.actions } : {}),
    ...(def.stores !== undefined ? { stores: def.stores } : {}),
    ...(def.effects !== undefined ? { effects: def.effects } : {}),
  };
}

/**
 * Declare the extra model fields a plugin contributes — type-only. The returned
 * value is a phantom (never read at runtime); actual values come from the
 * central session parse.
 *
 * @example
 *   schema: schema<{ user: { twoFactorEnabled: boolean } }>(),
 */
export function schema<T extends PluginSchema & Record<Exclude<keyof T, keyof PluginSchema>, never>>(): T {
  return undefined as unknown as T;
}

/**
 * Reject config keys a plugin does not declare. TypeScript skips
 * excess-property checking when the target is a type parameter, so a factory's
 * `const Config extends StrictConfig<Config, MyConfig>` has to do it.
 */
export type StrictConfig<Config, Allowed> = Allowed & Record<Exclude<keyof Config, keyof Allowed>, never>;

/** A plugin's model names mapped to the app columns declared on each. */
export type ModelFields = Record<string, object | undefined>;

/** The columns declared in a plugin's config, or none. */
export type DeclaredFields<Config> = Config extends { fields: infer M } ? M : unknown;

/** Build a plugin's `fields()` declaration function, constrained to its own models. */
export function defineFields<Allowed extends ModelFields>() {
  return <T extends Allowed & Record<Exclude<keyof T, keyof Allowed>, never>>(): T => undefined as unknown as T;
}
