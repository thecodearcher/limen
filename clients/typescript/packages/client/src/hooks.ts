import type {
  AfterResponseHook,
  BeforeRequestHook,
  PluginHooks,
  RequestContext,
  ResponseContext,
  RouteMatcher,
} from "./plugin";

type HookProvider = {
  hooks?: PluginHooks;
};

function matchesRoute(matcher: RouteMatcher | undefined, routePath: string): boolean {
  if (matcher === undefined) {
    return true;
  }
  if (typeof matcher === "string") {
    return routePath === matcher;
  }
  if (typeof matcher === "function") {
    return matcher({ path: routePath });
  }
  return routePath !== undefined && matcher.includes(routePath);
}

export class HookRunner {
  private readonly before: BeforeRequestHook[];
  private readonly after: AfterResponseHook[];

  constructor(plugins: readonly HookProvider[]) {
    this.before = plugins.flatMap((p) => p.hooks?.beforeRequest ?? []);
    this.after = plugins.flatMap((p) => p.hooks?.afterResponse ?? []);
  }

  async runBeforeRequest(initial: RequestContext): Promise<RequestContext> {
    let ctx = initial;
    for (const hook of this.before) {
      if (!matchesRoute(hook.match, ctx.routePath)) {
        continue;
      }
      ctx = await hook.run(ctx);
    }
    return ctx;
  }

  async runAfterResponse(initial: ResponseContext): Promise<ResponseContext> {
    let ctx = initial;
    for (const hook of this.after) {
      if (!matchesRoute(hook.match, ctx.routePath)) {
        continue;
      }
      if (!hook.allowOnFailure && ctx.status >= 400) {
        continue;
      }
      ctx = await hook.run(ctx);
    }
    return ctx;
  }
}
