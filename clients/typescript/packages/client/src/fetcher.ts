import { DEFAULT_TIMEOUT_MS } from "./constants";
import { unwrapErrorMessage, unwrapPayload } from "./envelope";
import { LimenError, resolveErrorCode } from "./errors";
import { ensureLeadingSlash, joinURL, stripTrailingSlash } from "./helpers";
import type { HookRunner } from "./hooks";
import type { FetchInit, FetchOptions, RequestContext, ResponseContext } from "./plugin";
import type { EnvelopeConfig, QueryParams } from "./types";

export type FetcherFetchOptions = FetchOptions & {
  /** Whether to send credentials (cookies). Defaults to `"include"`. */
  credentials?: RequestCredentials;
  /** Custom fetch impl. Defaults to `globalThis.fetch`. */
  impl?: typeof fetch;
  /** Callback function to be called when the request is successful. */
  onSuccess?: (context: ResponseContext & { response: Response }) => void;
  /** Callback function to be called when the request fails. */
  onError?: (context: ResponseContext & { error: Error }) => void;
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
    const callInit = init ?? {};
    const reqCtx = await this.prepareRequest(path, callInit, routePath);

    let response: Response;
    try {
      response = await this.fetchImpl(reqCtx.url, this.buildRequestInit(reqCtx, callInit.timeout));
    } catch (err) {
      this.fail(reqCtx, 0, toRequestError(err));
    }

    return this.handleResponse<T>(reqCtx, response);
  }

  private async prepareRequest(path: string, init: FetchInit, routePath: string): Promise<RequestContext> {
    const fullPath = this.normalizeRelativePath(this.opts.basePath, path, init.query);
    const reqCtx: RequestContext = {
      method: init.method ?? (init.body !== undefined ? "POST" : "GET"),
      fullPath,
      path,
      routePath,
      url: joinURL(this.opts.baseURL, fullPath),
      headers: this.buildHeaders(init.headers),
      body: init.body,
    };
    return this.opts.hooks.runBeforeRequest(reqCtx);
  }

  private buildHeaders(extra: HeadersInit | undefined): Headers {
    const headers = new Headers(this.opts.fetchOptions.headers);
    new Headers(extra).forEach((value, key) => headers.set(key, value));

    if (!headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    if (!headers.has("Accept")) {
      headers.set("Accept", "application/json");
    }
    return headers;
  }

  private buildRequestInit(reqCtx: RequestContext, timeout: number | undefined): RequestInit {
    const requestInit: RequestInit = {
      method: reqCtx.method,
      headers: reqCtx.headers,
      credentials: this.credentials,
    };

    if (reqCtx.body !== undefined && reqCtx.body !== null) {
      requestInit.body = JSON.stringify(reqCtx.body);
    }

    const signal = this.timeoutSignal(timeout);
    if (signal !== undefined) {
      requestInit.signal = signal;
    }

    return requestInit;
  }

  private async handleResponse<T>(reqCtx: RequestContext, response: Response): Promise<T> {
    // A non-JSON body on a 2xx response is treated as a failure.
    let parsedBody: unknown;
    let parseFailed = false;
    try {
      parsedBody = await this.parseResponseBody(response);
    } catch {
      parseFailed = true;
    }

    const ok = response.ok && !parseFailed;
    const unwrapped = ok && parsedBody !== undefined ? unwrapPayload(parsedBody, this.opts.envelope) : parsedBody;

    const resCtx = await this.opts.hooks.runAfterResponse({
      method: reqCtx.method,
      fullPath: reqCtx.fullPath,
      path: reqCtx.path,
      routePath: reqCtx.routePath,
      status: response.status,
      ok,
      headers: response.headers,
      body: unwrapped,
    });

    if (resCtx.ok) {
      this.opts.fetchOptions.onSuccess?.({ ...resCtx, response });
      return resCtx.body as T;
    }

    const message =
      parseFailed && response.ok
        ? "Invalid JSON in response body"
        : (unwrapErrorMessage(resCtx.body, this.opts.envelope) ??
          response.statusText ??
          `Request failed with status ${response.status}`);
    this.fail(reqCtx, response.status, new LimenError(message, response.status, resolveErrorCode(response.status, resCtx.body)));
  }

  private fail(reqCtx: RequestContext, status: number, error: LimenError): never {
    this.opts.fetchOptions.onError?.({ ...reqCtx, status, ok: false, error });
    throw error;
  }

  private timeoutSignal(perCallTimeout: number | undefined): AbortSignal | undefined {
    const timeoutMs = perCallTimeout ?? this.opts.fetchOptions.timeout ?? DEFAULT_TIMEOUT_MS;
    if (timeoutMs <= 0 || typeof AbortSignal === "undefined" || typeof AbortSignal.timeout !== "function") {
      return undefined;
    }
    return AbortSignal.timeout(timeoutMs);
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

  private normalizeRelativePath(basePath: string, relativePath: string, query: QueryParams | undefined): string {
    const base = basePath === "" || basePath === "/" ? "" : ensureLeadingSlash(stripTrailingSlash(basePath));
    let path = base + ensureLeadingSlash(relativePath);
    if (query !== undefined) {
      const qs = new URLSearchParams(query).toString();
      if (qs.length > 0) {
        path = `${path}?${qs}`;
      }
    }
    return path;
  }
}

function toRequestError(err: unknown): LimenError {
  if (err instanceof Error) {
    if (err.name === "TimeoutError" || err.name === "AbortError") {
      return new LimenError("Request timed out", 0, "timeout");
    }
  }
  return new LimenError((err as Error)?.message ?? "Network request failed", 0, "unknown");
}
