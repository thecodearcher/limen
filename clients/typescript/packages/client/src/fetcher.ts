import { unwrapErrorMessage, unwrapPayload } from "./envelope";
import { LimenError, deriveErrorCode } from "./errors";
import { ensureLeadingSlash, joinURL, stripTrailingSlash } from "./helpers";
import type { HookRunner } from "./hooks";
import type { FetchInit, RequestContext, ResponseContext } from "./plugin";
import type { EnvelopeConfig, HTTPMethod } from "./types";

type ClientFetchCallbackContext = ResponseContext & {};

export type FetcherFetchOptions = {
  /** Whether to send credentials (cookies). Defaults to `"include"`. */
  credentials?: RequestCredentials;
  /** Custom fetch impl. Defaults to `globalThis.fetch`. */
  impl?: typeof fetch;
  /** Default headers merged into every request. Per-request headers override these. */
  headers?: HeadersInit;
  /** Callback function to be called when the request is successful. */
  onSuccess?: (context: ClientFetchCallbackContext & { response: Response }) => void;
  /** Callback function to be called when the request fails. */
  onError?: (context: ClientFetchCallbackContext & { error: Error }) => void;
};

type FetcherOptions = {
  baseURL: string;
  basePath: string;
  fetchOptions: FetcherFetchOptions;
  hooks: HookRunner;
  envelope: EnvelopeConfig;
};

export class Fetcher {
  private readonly fetchImpl: typeof fetch;
  private readonly credentials: RequestCredentials;

  constructor(private readonly opts: FetcherOptions) {
    this.fetchImpl = opts.fetchOptions.impl ?? globalThis.fetch.bind(globalThis);
    this.credentials = opts.fetchOptions.credentials ?? "include";
  }

  async fetch<T>(path: string, init?: FetchInit, routePath: string = path): Promise<T> {
    const method: HTTPMethod = init?.method ?? (init?.body !== undefined ? "POST" : "GET");
    return this.run<T>({
      method,
      path,
      routePath,
      body: init?.body,
      headers: init?.headers,
      query: init?.query,
    });
  }

  /**
   * Builds the request context, runs hooks, does
   * the fetch, parses the response, runs hooks again, throws on non-2xx.
   */
  private async run<T>(args: {
    method: HTTPMethod;
    path: string;
    routePath: string;
    body: unknown;
    headers: HeadersInit | undefined;
    query: Record<string, string> | undefined;
  }): Promise<T> {
    const fullPath = this.normalizeRelativePath(this.opts.basePath, args.path, args.query);
    const url = joinURL(this.opts.baseURL, fullPath);

    const headers = new Headers({ ...args.headers, ...(this.opts.fetchOptions.headers ?? {}) });
    if (!headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    if (!headers.has("Accept")) {
      headers.set("Accept", "application/json");
    }

    let reqCtx: RequestContext = {
      method: args.method,
      fullPath: fullPath,
      path: args.path,
      routePath: args.routePath,
      url,
      headers,
      body: args.body,
    };

    reqCtx = await this.opts.hooks.runBeforeRequest(reqCtx);

    const payload = reqCtx.body !== undefined && reqCtx.body !== null ? JSON.stringify(reqCtx.body) : undefined;

    const requestInit: RequestInit = {
      method: reqCtx.method,
      headers: reqCtx.headers,
      credentials: this.credentials,
    };

    if (payload !== undefined) {
      requestInit.body = payload;
    }

    let response: Response;
    try {
      response = await this.fetchImpl(reqCtx.url, requestInit);
    } catch (err) {
      this.opts.fetchOptions.onError?.({
        ...reqCtx,
        status: 0,
        ok: false,
        error: err as Error,
      });
      throw new LimenError(err instanceof Error ? err.message : "Network request failed", 0, "unknown");
    }

    const parsedBody = await this.parseResponseBody(response);
    const unwrapped =
      response.ok && parsedBody !== undefined ? unwrapPayload(parsedBody, this.opts.envelope) : parsedBody;

    let resCtx: ResponseContext = {
      method: args.method,
      fullPath: fullPath,
      path: args.path,
      routePath: args.routePath,
      status: response.status,
      ok: response.ok,
      headers: response.headers,
      body: unwrapped,
    };

    resCtx = await this.opts.hooks.runAfterResponse(resCtx);

    if (resCtx.ok) {
      this.opts.fetchOptions.onSuccess?.({
        ...resCtx,
        response,
      });
      return resCtx.body as T;
    }

    const message =
      unwrapErrorMessage(resCtx.body, this.opts.envelope) ??
      response.statusText ??
      `Request failed with status ${response.status}`;

    const error = new LimenError(message, response.status, deriveErrorCode(response.status));
    this.opts.fetchOptions.onError?.({
      ...reqCtx,
      status: response.status,
      ok: false,
      error,
    });
    throw error;
  }

  private async parseResponseBody(response: Response): Promise<unknown> {
    if (response.status === 204) {
      return undefined;
    }
    const text = await response.text();
    if (text.length === 0) {
      return undefined;
    }
    return JSON.parse(text) as unknown;
  }

  private normalizeRelativePath(
    basePath: string,
    relativePath: string,
    query: Record<string, string> | undefined,
  ): string {
    const base = basePath === "" || basePath === "/" ? "" : ensureLeadingSlash(stripTrailingSlash(basePath));
    let path = base + ensureLeadingSlash(relativePath);
    if (query !== undefined) {
      const params = new URLSearchParams(query);
      const qs = params.toString();
      if (qs.length > 0) {
        path = `${path}?${qs}`;
      }
    }
    return path;
  }
}
