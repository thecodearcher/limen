import type { AnyRouteContext } from "./context";
import type { LimenError } from "./errors";
import { camelizeEach, camelizeKeys, camelizePage, isPageResponse } from "./helpers";
import { resolvePath } from "./path";
import type { FetchInit, FetchOptions } from "./plugin";
import type { AnyRoute, HttpRunner, RouteCallOptions } from "./route";
import { defaultSerialize } from "./serialize";
import type { QueryParams, Session } from "./types";

/**
 * Run the default HTTP steps for a route — merge defaults, resolve path params,
 * serialize, dispatch, parse — without applying session effects.
 * `onSession` receives a session `parseSession` accepted. Handlers omit it.
 */
async function runHttp(
  ctx: AnyRouteContext,
  def: AnyRoute,
  input: unknown,
  callInit?: FetchOptions,
  onSession?: (session: Session<unknown>) => void,
): Promise<unknown> {
  let merged = input;
  if (def.defaults !== undefined) {
    merged = { ...(def.defaults as Record<string, unknown>), ...((input ?? {}) as Record<string, unknown>) };
  }

  const { path, rest } = resolvePath(def.path, def.params, merged);
  const payload = def.serialize !== undefined ? def.serialize(rest) : defaultSerialize(rest);

  const init: FetchInit = { ...callInit, method: def.method, absolute: def.absolute ?? false };
  if (def.method === "GET" && payload !== undefined) {
    init.query = payload as QueryParams;
  } else {
    init.body = payload;
  }

  const raw = await ctx.fetch<unknown>(path, init);

  if (def.parseSession === true) {
    const session = ctx.parseSession(raw);
    if (session !== false) {
      onSession?.(session);
      return session;
    }
  }

  if (def.parse !== undefined) {
    return def.parse(raw);
  }
  if (isPageResponse(raw)) {
    return camelizePage(raw);
  }
  return Array.isArray(raw) ? camelizeEach(raw) : camelizeKeys(raw);
}

async function applyEffects(
  ctx: AnyRouteContext,
  def: AnyRoute,
  session: Session<unknown> | undefined,
): Promise<void> {
  if (def.clearSession === true) {
    ctx.store.setData(null);
  }

  if (session !== undefined && def.skipStore !== true) {
    ctx.store.setData(session);
  }

  if (def.refetchSession === true) {
    await ctx.store.refetch();
  }
}

function makeHttpRunner(
  ctx: AnyRouteContext,
  def: AnyRoute,
  boundInput: unknown,
  callInit?: FetchOptions,
): HttpRunner<unknown> {
  const run = (override?: unknown): Promise<unknown> =>
    runHttp(ctx, def, override === undefined ? boundInput : override, callInit);
  return run as HttpRunner<unknown>;
}

/**
 * Execute a route's behaviour: delegate to its `handler` when present (handler
 * owns all behaviour, including any effects), otherwise run the default
 * pipeline and apply declarative effects once at the top level.
 */
async function dispatchRoute(
  ctx: AnyRouteContext,
  def: AnyRoute,
  input: unknown,
  callInit?: FetchOptions,
): Promise<unknown> {
  if (def.handler !== undefined) {
    return def.handler(ctx, input, makeHttpRunner(ctx, def, input, callInit));
  }
  let session: Session<unknown> | undefined;
  const result = await runHttp(ctx, def, input, callInit, (parsed) => {
    session = parsed;
  });
  await applyEffects(ctx, def, session);
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
  const { onSuccess, onError, ...callInit }: RouteCallOptions = opts ?? {};
  try {
    const result = await dispatchRoute(ctx, def, input, callInit);
    onSuccess?.(result);
    return result;
  } catch (error) {
    onError?.(error as LimenError);
    throw error;
  }
}
