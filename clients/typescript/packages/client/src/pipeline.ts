import type { AnyRouteContext } from "./context";
import type { LimenError } from "./errors";
import { camelizeEach } from "./helpers";
import { resolvePath } from "./path";
import type { FetchInit } from "./plugin";
import type { AnyRoute, HttpRunner, RouteCallOptions } from "./route";
import { defaultSerialize } from "./serialize";
import type { Session } from "./types";

/**
 * Run the default HTTP steps for a route — merge defaults, resolve path params,
 * serialize, dispatch, parse — without applying session effects.
 */
async function runHttp(ctx: AnyRouteContext, def: AnyRoute, input: unknown): Promise<unknown> {
  let merged = input;
  if (def.defaults !== undefined) {
    merged = { ...(def.defaults as Record<string, unknown>), ...((input ?? {}) as Record<string, unknown>) };
  }

  const { path, rest } = resolvePath(def.path, def.params, merged);
  const payload = def.serialize !== undefined ? def.serialize(rest) : defaultSerialize(rest);

  const init: FetchInit = { method: def.method, absolute: def.absolute ?? false };
  if (def.method === "GET" && payload !== undefined) {
    init.query = payload as Record<string, string>;
  } else {
    init.body = payload;
  }

  const raw = await ctx.fetch<unknown>(path, init);

  if (def.parseSession === true) {
    return ctx.parseSession(raw);
  }

  if (def.parse !== undefined) {
    return def.parse(raw);
  }
  return Array.isArray(raw) ? camelizeEach(raw) : raw;
}

async function applyEffects(ctx: AnyRouteContext, def: AnyRoute, result: unknown): Promise<void> {
  if (def.clearSession === true) {
    ctx.store.setData(null);
  }

  if (def.parseSession === true && def.skipStore !== true) {
    if (result !== null && typeof result === "object" && "user" in result) {
      ctx.store.setData(result as Session<unknown>);
    }
  }

  if (def.refetchSession === true) {
    await ctx.store.refetch();
  }
}

function makeHttpRunner(ctx: AnyRouteContext, def: AnyRoute, boundInput: unknown): HttpRunner<unknown> {
  const run = (override?: unknown): Promise<unknown> =>
    runHttp(ctx, def, override === undefined ? boundInput : override);
  return run as HttpRunner<unknown>;
}

/**
 * Execute a route's behaviour: delegate to its `handler` when present (handler
 * owns all behaviour, including any effects), otherwise run the default
 * pipeline and apply declarative effects once at the top level.
 */
async function dispatchRoute(ctx: AnyRouteContext, def: AnyRoute, input: unknown): Promise<unknown> {
  if (def.handler !== undefined) {
    return def.handler(ctx, input, makeHttpRunner(ctx, def, input));
  }
  const result = await runHttp(ctx, def, input);
  await applyEffects(ctx, def, result);
  return result;
}

/**
 * Run a route as a public client call, firing the per-call `onSuccess` /
 * `onError` hooks around the resolved value or thrown error.
 */
export async function runRoute(
  ctx: AnyRouteContext,
  def: AnyRoute,
  input: unknown,
  opts?: RouteCallOptions,
): Promise<unknown> {
  try {
    const result = await dispatchRoute(ctx, def, input);
    opts?.onSuccess?.(result);
    return result;
  } catch (error) {
    opts?.onError?.(error as LimenError);
    throw error;
  }
}
