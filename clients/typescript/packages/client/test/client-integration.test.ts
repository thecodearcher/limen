import { describe, expect, it, vi } from "vitest";
import { createAuthClient, LimenError } from "../src";
import { bearerPlugin } from "../src/plugins/bearer";
import { credentialPasswordPlugin } from "../src/plugins/credential";
import { magicLinkPlugin } from "../src/plugins/magic-link";
import { oauthClientPlugin } from "../src/plugins/oauth";
import { twoFactorPlugin } from "../src/plugins/two-factor";

type Recorded = { url: string; method: string; body: unknown; headers: Headers };
type MockReply = { status?: number; body?: unknown; headers?: Record<string, string> };

function mockFetch(reply: (req: Recorded) => MockReply) {
  const calls: Recorded[] = [];
  const impl = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const rawBody = init?.body;
    const rec: Recorded = {
      url,
      method: init?.method ?? "GET",
      body: typeof rawBody === "string" && rawBody.length > 0 ? JSON.parse(rawBody) : undefined,
      headers: new Headers(init?.headers),
    };
    calls.push(rec);
    const out = reply(rec);
    const payload = out.body === undefined ? "" : JSON.stringify(out.body);
    const responseInit: ResponseInit = { status: out.status ?? 200 };
    if (out.headers) {
      responseInit.headers = out.headers;
    }
    return new Response(payload, responseInit);
  }) as typeof fetch;
  return { impl, calls };
}

const userBody = { user: { id: "u1", email: "ada@example.com", email_verified_at: null, first_name: "Ada" } };

function setup(reply: (req: Recorded) => MockReply, redirectFn?: (url: string) => boolean) {
  const { impl, calls } = mockFetch(reply);
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [
      credentialPasswordPlugin(),
      magicLinkPlugin(),
      oauthClientPlugin(),
      twoFactorPlugin({ onTwoFactorRedirect: () => {} }),
      bearerPlugin(),
    ],
    fetchOptions: { impl },
    crossTabSync: false,
    refetchOnWindowFocus: false,
    ...(redirectFn ? { redirectFn } : {}),
  });
  return { auth, calls };
}

describe("createAuthClient — request shaping", () => {
  it("signIn.credential: POST under basePath, camel→snake body, rememberMe default, persists session", async () => {
    const { auth, calls } = setup(() => ({ body: userBody }));

    const session = await auth.signIn.credential({ credential: "ada@example.com", password: "pw" });

    expect(calls[0]?.method).toBe("POST");
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/signin/credential");
    expect(calls[0]?.body).toEqual({ credential: "ada@example.com", password: "pw", remember_me: true });
    expect(session.user.id).toBe("u1");
    expect((session.user as { firstName?: string }).firstName).toBe("Ada");
    expect(auth.$session.get().data?.user.id).toBe("u1");
  });

  it("getSession(): GET /me, reconciles the store, returns the session", async () => {
    const { auth, calls } = setup(() => ({ body: userBody }));
    const session = await auth.getSession();
    expect(calls[0]?.method).toBe("GET");
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/me");
    expect(session?.user.id).toBe("u1");
    expect(auth.$session.get().data?.user.id).toBe("u1");
  });

  it("signout(): POST /signout then clears the session", async () => {
    const { auth, calls } = setup(() => ({ body: userBody }));
    await auth.signIn.credential({ credential: "ada@example.com", password: "pw" });
    expect(auth.$session.get().data).not.toBeNull();

    await auth.signout();
    expect(calls.at(-1)?.method).toBe("POST");
    expect(calls.at(-1)?.url).toBe("http://localhost:8080/auth/signout");
    expect(auth.$session.get().data).toBeNull();
  });

  it("verifyEmail(): POST /verify-email then refetches /me", async () => {
    const { auth, calls } = setup((req) => (req.url.endsWith("/verify-email") ? { body: "ok" } : { body: userBody }));
    const message = await auth.verifyEmail({ token: "tok" });
    expect(message).toBe("ok");
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/verify-email");
    expect(calls[0]?.body).toEqual({ token: "tok" });
    expect(calls[1]?.method).toBe("GET");
    expect(calls[1]?.url).toBe("http://localhost:8080/auth/me");
  });

  it("username.checkAvailability(): unwraps `{ available }` to a boolean", async () => {
    const { auth, calls } = setup(() => ({ body: { available: false } }));
    const available = await auth.username.checkAvailability({ username: "ada" });
    expect(available).toBe(false);
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/usernames/check");
    expect(calls[0]?.body).toEqual({ username: "ada" });
  });
});

describe("createAuthClient — oauth path params + redirect handler", () => {
  it("signIn.social(): substitutes :provider, strips provider + disableRedirect, no auto-nav when disabled", async () => {
    const { auth, calls } = setup(() => ({ body: { url: "https://accounts.google.com/o/oauth2/auth?x=1" } }));

    const result = await auth.signIn.social({
      provider: "google",
      redirectUri: "https://app/cb",
      disableRedirect: true,
    });

    expect(calls[0]?.method).toBe("GET");
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/oauth/google/authorize?redirect_uri=https%3A%2F%2Fapp%2Fcb");
    expect(calls[0]?.url).not.toContain("provider");
    expect(calls[0]?.url).not.toContain("disable");
    expect(result).toEqual({ url: "https://accounts.google.com/o/oauth2/auth?x=1", redirect: false });
  });

  it("signIn.social(): navigates via the configured redirectFn by default", async () => {
    const redirectFn = vi.fn(() => true);
    const { auth } = setup(() => ({ body: { url: "https://provider/auth" } }), redirectFn);

    const result = await auth.signIn.social({ provider: "github" });
    expect(redirectFn).toHaveBeenCalledWith("https://provider/auth");
    expect(result.redirect).toBe(true);
  });

  it("social.tokens()/social.refreshTokens(): provider in path, camelized response", async () => {
    const { auth, calls } = setup(() => ({ body: { access_token: "at", refresh_token: "rt" } }));

    const tokens = await auth.social.tokens({ provider: "google" });
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/oauth/google/tokens");
    expect(tokens).toEqual({ accessToken: "at", refreshToken: "rt" });

    await auth.social.refreshTokens({ provider: "google" });
    expect(calls[1]?.method).toBe("POST");
    expect(calls[1]?.url).toBe("http://localhost:8080/auth/oauth/google/tokens/refresh");
  });

  it("social.accounts(): array response is camelized by the DEFAULT parse (no explicit `parse`)", async () => {
    const { auth, calls } = setup(() => ({
      body: [{ provider: "google", provider_account_id: "u-1", scopes: ["email"], created_at: "t0", updated_at: "t1" }],
    }));
    const accounts = await auth.social.accounts();
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/oauth/accounts");
    expect(accounts).toEqual([
      { provider: "google", providerAccountId: "u-1", scopes: ["email"], createdAt: "t0", updatedAt: "t1" },
    ]);
  });
});

describe("createAuthClient — two-factor as-pinned routes", () => {
  it("getTotpUri(): GET /two-factor/totp/uri", async () => {
    const { auth, calls } = setup(() => ({ body: { uri: "otpauth://x" } }));
    const res = await auth.twoFactor.getTotpUri();
    expect(res).toEqual({ uri: "otpauth://x" });
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/two-factor/totp/uri");
  });

  it("getBackupCodes() is GET and regenerateBackupCodes() is PUT on the same path", async () => {
    const { auth, calls } = setup(() => ({ body: ["a", "b"] }));
    await auth.twoFactor.getBackupCodes();
    await auth.twoFactor.regenerateBackupCodes();
    expect(calls[0]?.method).toBe("GET");
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/two-factor/backup-codes");
    expect(calls[1]?.method).toBe("PUT");
    expect(calls[1]?.url).toBe("http://localhost:8080/auth/two-factor/backup-codes");
  });
});

describe("createAuthClient — hooks + overrides", () => {
  it("bearer plugin injects the stored access token on later requests", async () => {
    const { auth, calls } = setup(() => ({ body: userBody }));
    auth.bearer.setTokens({ accessToken: "secret-token" });
    await auth.getSession();
    expect(calls[0]?.headers.get("Authorization")).toBe("Bearer secret-token");
  });

  it("client `overrides` remount a plugin's base path", async () => {
    const { impl, calls } = mockFetch(() => ({ body: { message: "sent" } }));
    const auth = createAuthClient({
      baseURL: "http://localhost:8080",
      plugins: [magicLinkPlugin()],
      overrides: { magicLink: { basePath: "/passwordless" } },
      fetchOptions: { impl },
      crossTabSync: false,
      refetchOnWindowFocus: false,
    });
    await auth.magicLink.signin({ email: "ada@example.com" });
    expect(calls[0]?.url).toBe("http://localhost:8080/auth/passwordless/signin");
  });
});

describe("createAuthClient — per-call options", () => {
  it("onSuccess fires with the resolved value", async () => {
    const { auth } = setup(() => ({ body: userBody }));
    const onSuccess = vi.fn();
    const session = await auth.signIn.credential({ credential: "ada@example.com", password: "pw" }, { onSuccess });
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledWith(session);
    expect(session.user.id).toBe("u1");
  });

  it("onError fires with the LimenError and the call still rejects", async () => {
    const { auth } = setup(() => ({ status: 500, body: { message: "boom" } }));
    const onError = vi.fn();
    await expect(auth.sessions(undefined, { onError })).rejects.toBeInstanceOf(LimenError);
    expect(onError).toHaveBeenCalledTimes(1);
    expect(onError.mock.calls[0]?.[0]).toBeInstanceOf(LimenError);
  });
});

describe("createAuthClient — assembly", () => {
  it("merges contributions from multiple plugins (including the shared signIn namespace)", () => {
    const { auth } = setup(() => ({ body: userBody }));
    expect(typeof auth.signIn.credential).toBe("function");
    expect(typeof auth.signIn.social).toBe("function");
    expect(typeof auth.password.set).toBe("function");
    expect(typeof auth.getSession).toBe("function");
    expect(typeof auth.sessions).toBe("function");
    expect(typeof auth.social.unlink).toBe("function");
    expect(typeof auth.twoFactor.sendOTP).toBe("function");
    expect(typeof auth.bearer.getTokens).toBe("function");
  });
});
