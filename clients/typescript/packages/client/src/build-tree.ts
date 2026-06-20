import type { AnyRouteContext, RouteContext } from "./context";
import type { AnyClientPlugin, RunRoute } from "./define-plugin";
import type { Fetcher } from "./fetcher";
import { ensureLeadingSlash, kebabToCamel, normalizeBasePath } from "./helpers";
import { chainFromDotted, pathToChain } from "./path";
import { runRoute } from "./pipeline";
import type { FetchInit } from "./plugin";
import type { AnyRoute, AnyRouteDescriptor, RouteCallOptions } from "./route";

export type ClientOverrides = Record<string, { basePath?: string } | undefined> | undefined;

/** Scope a context's `fetch` to one plugin's base path (after client `overrides`). */
function scopeContext(
  ctx: AnyRouteContext,
  fetcher: Fetcher,
  plugin: AnyClientPlugin,
  overrides: ClientOverrides,
): RouteContext {
  const defaultBase = normalizeBasePath(plugin.basePath ?? "");
  const overrideBase = overrides?.[kebabToCamel(plugin.id)]?.basePath;
  const resolvedBase = normalizeBasePath(overrideBase ?? plugin.basePath ?? "");
  return {
    ...ctx,
    fetch: <T>(path: string, init?: FetchInit) => {
      const absolute = init?.absolute === true;
      const requestPath = (absolute ? "" : resolvedBase) + ensureLeadingSlash(path);
      const routePath = (absolute ? "" : defaultBase) + ensureLeadingSlash(path);
      return fetcher.fetch<T>(requestPath, init, routePath);
    },
  };
}

function chainFor(plugin: AnyClientPlugin, def: AnyRoute): string[] {
  if (typeof def.as === "string") {
    return chainFromDotted(def.as);
  }
  return [...pathToChain(plugin.basePath ?? ""), ...pathToChain(def.path)];
}

function mountAtChain(target: Record<string, unknown>, pathSegments: string[], callable: unknown): void {
  let current = target;
  for (let i = 0; i < pathSegments.length - 1; i += 1) {
    const segment = pathSegments[i] as string;
    const child = current[segment];
    if (child === undefined) {
      const namespace: Record<string, unknown> = {};
      current[segment] = namespace;
      current = namespace;
    } else {
      current = child as Record<string, unknown>;
    }
  }
  const finalSegment = pathSegments[pathSegments.length - 1] as string;
  current[finalSegment] = callable;
}

function isNamespace(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value) && typeof value !== "function";
}

function mergeInto(target: Record<string, unknown>, source: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(source)) {
    const existing = target[key];
    if (isNamespace(existing) && isNamespace(value)) {
      mergeInto(existing, value);
      continue;
    }
    target[key] = value;
  }
}

type BuildClientTreeArgs = {
  plugins: readonly AnyClientPlugin[];
  ctx: AnyRouteContext;
  fetcher: Fetcher;
  overrides: ClientOverrides;
};

/**
 * Build the public API object from plugin routes and actions.
 */
export function buildClientTree({ plugins, ctx, fetcher, overrides }: BuildClientTreeArgs): Record<string, unknown> {
  const api: Record<string, unknown> = {};

  for (const plugin of plugins) {
    const scopedCtx = scopeContext(ctx, fetcher, plugin, overrides);
    const contribution: Record<string, unknown> = {};

    for (const def of plugin.routes as readonly AnyRoute[]) {
      if (def.expose === false) {
        continue;
      }
      const call = (input?: unknown, opts?: RouteCallOptions) => runRoute(scopedCtx, def, input, opts);
      mountAtChain(contribution, chainFor(plugin, def), call);
    }

    if (plugin.actions !== undefined) {
      const run: RunRoute = (route, input) => runRoute(scopedCtx, route as AnyRouteDescriptor, input) as Promise<never>;
      mergeInto(contribution, plugin.actions(scopedCtx, run) as Record<string, unknown>);
    }

    mergeInto(api, contribution);
  }

  return api;
}
