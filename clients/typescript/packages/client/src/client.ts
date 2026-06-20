import type { ClientOverrides } from "./build-tree";
import { buildClientTree } from "./build-tree";
import { DEFAULT_ENVELOPE_CONFIG } from "./constants";
import type { RouteContext } from "./context";
import type { AnyClientPlugin } from "./define-plugin";
import { Fetcher } from "./fetcher";
import { normalizeBasePath, stripTrailingSlash } from "./helpers";
import { HookRunner } from "./hooks";
import { defaultSessionParse } from "./normalize";
import type { FetchInit } from "./plugin";
import { coreClientPlugin, createSessionHydrator } from "./routes";
import { createSessionStore } from "./session-store";
import type { AuthClient, ClientFetchOptions, CreateAuthClientOptions, EnvelopeConfig, RedirectFn } from "./types";

export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): AuthClient<Plugins, TFields> {
  const baseURL = stripTrailingSlash(opts.baseURL);
  const basePath = normalizeBasePath(opts.basePath ?? "/auth");

  const userPlugins = (opts.plugins ?? []) as readonly AnyClientPlugin[];
  const plugins: readonly AnyClientPlugin[] = [coreClientPlugin<TFields>(), ...userPlugins];

  const hooks = new HookRunner(plugins);
  const envelope = { ...DEFAULT_ENVELOPE_CONFIG, ...opts.envelope } satisfies EnvelopeConfig;
  const fetcher = buildFetcher(baseURL, basePath, envelope, hooks, opts.fetchOptions ?? {});

  const parseSession = opts.parseSession ?? defaultSessionParse;
  const redirect = resolveRedirect(opts.redirectFn);
  const baseFetch = <T>(path: string, init?: FetchInit) => fetcher.fetch<T>(path, init);

  const store = createSessionStore<TFields>({
    hydrator: createSessionHydrator<TFields>({ fetch: baseFetch, parseSession }),
    crossTabSync: opts.crossTabSync !== false,
    refetchOnWindowFocus: opts.refetchOnWindowFocus !== false,
    ...(opts.initialSession !== undefined ? { initialSession: opts.initialSession } : {}),
  });

  const ctx: RouteContext<TFields> = {
    fetch: baseFetch,
    redirect,
    parseSession,
    setSession: (session) => store.setData(session),
    refetchSession: () => store.refetch(),
    store,
  };

  const api = buildClientTree({
    plugins,
    ctx,
    fetcher,
    overrides: opts.overrides as ClientOverrides,
  });

  const client: Record<string, unknown> = {
    baseURL,
    basePath,
    ...api,
    $session: store.$session,
  };

  return client as AuthClient<Plugins, TFields>;
}

function buildFetcher(
  baseURL: string,
  basePath: string,
  envelope: EnvelopeConfig,
  hooks: HookRunner,
  fetchOptions: ClientFetchOptions,
): Fetcher {
  return new Fetcher({
    baseURL,
    basePath,
    envelope,
    hooks,
    fetchOptions,
  });
}

function resolveRedirect(redirect: RedirectFn | undefined): RedirectFn {
  return (url: string) => {
    if (redirect !== undefined) {
      return redirect(url);
    }
    if (typeof window !== "undefined" && typeof window.location !== "undefined") {
      window.location.href = url;
      return true;
    }
    return false;
  };
}
