import { describe, expect, it, vi } from "vitest";
import { createAuthClient, LimenError } from "../src";

/** A fetch impl that never resolves but rejects when its signal aborts — like real fetch. */
function hangingImpl(): typeof fetch {
  return ((_url: RequestInfo | URL, init?: RequestInit) =>
    new Promise<Response>((_resolve, reject) => {
      const signal = init?.signal;
      if (!signal) {
        return;
      }
      if (signal.aborted) {
        reject(signal.reason as Error);
        return;
      }
      signal.addEventListener("abort", () => reject(signal.reason as Error), { once: true });
    })) as typeof fetch;
}

/** A fetch impl returning one fixed response. */
function staticImpl(body: string, init?: ResponseInit): typeof fetch {
  return (async () => new Response(body, init)) as typeof fetch;
}

function makeClient(impl: typeof fetch, fetchOptions: Record<string, unknown> = {}) {
  return createAuthClient({
    baseURL: "http://localhost:8080",
    fetchOptions: { impl, ...fetchOptions },
    crossTabSync: false,
    refetchOnWindowFocus: false,
  });
}

describe("fetcher — timeout", () => {
  it("rejects with a timeout LimenError when the client timeout elapses", async () => {
    const auth = makeClient(hangingImpl(), { timeout: 20 });
    const err = (await auth.sessions().catch((e) => e)) as LimenError;
    expect(err).toBeInstanceOf(LimenError);
    expect(err.code).toBe("timeout");
    expect(err.is("timeout")).toBe(true);
  });

  it("per-call timeout overrides the client default", async () => {
    const auth = makeClient(hangingImpl(), { timeout: 0 }); // client default disabled
    const err = (await auth.sessions(undefined, { timeout: 20 }).catch((e) => e)) as LimenError;
    expect(err).toBeInstanceOf(LimenError);
    expect(err.is("timeout")).toBe(true);
  });
});

describe("fetcher — error codes", () => {
  it("uses a known server code instead of the status-derived code", async () => {
    const auth = makeClient(
      staticImpl(JSON.stringify({ message: "email is not verified", code: "email_not_verified" }), { status: 403 }),
      { timeout: 0 },
    );
    const err = (await auth.sessions().catch((e) => e)) as LimenError;
    expect(err).toBeInstanceOf(LimenError);
    expect(err.status).toBe(403);
    expect(err.code).toBe("email_not_verified");
  });

  it("derives forbidden when a 403 body has no code", async () => {
    const auth = makeClient(staticImpl(JSON.stringify({ message: "nope" }), { status: 403 }), { timeout: 0 });
    const err = (await auth.sessions().catch((e) => e)) as LimenError;
    expect(err.code).toBe("forbidden");
  });
});

describe("fetcher — non-JSON body", () => {
  it("wraps a non-JSON 2xx body in a LimenError and fires onError", async () => {
    const onError = vi.fn();
    const auth = makeClient(staticImpl("<html>not json</html>", { status: 200 }), { timeout: 0, onError });
    const err = (await auth.sessions().catch((e) => e)) as LimenError;
    expect(err).toBeInstanceOf(LimenError);
    expect(err.message).toMatch(/json/i);
    expect(onError).toHaveBeenCalledTimes(1);
    expect(onError.mock.calls[0]?.[0]?.error).toBeInstanceOf(LimenError);
  });
});

describe("fetcher — per-call init", () => {
  it("forwards per-call headers (and any FetchInit option) to the request", async () => {
    let seen: Headers | undefined;
    const impl = ((_url: RequestInfo | URL, init?: RequestInit) => {
      seen = new Headers(init?.headers);
      return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
    }) as typeof fetch;
    const auth = makeClient(impl, { timeout: 0 });

    await auth.sessions(undefined, { headers: { "X-Test": "1" } });

    expect(seen?.get("X-Test")).toBe("1");
  });

  it("per-call headers override client-default headers", async () => {
    let seen: Headers | undefined;
    const impl = ((_url: RequestInfo | URL, init?: RequestInit) => {
      seen = new Headers(init?.headers);
      return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
    }) as typeof fetch;
    const auth = makeClient(impl, { timeout: 0, headers: { "X-Test": "default" } });

    await auth.sessions(undefined, { headers: { "X-Test": "override" } });

    expect(seen?.get("X-Test")).toBe("override");
  });
});
