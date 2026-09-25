import { describe, expect, it, vi } from "vitest";
import { createAuthClient, LimenError } from "../src";
import { bearerPlugin } from "../src/plugins/bearer";
import { credentialPasswordPlugin } from "../src/plugins/credential";
import { magicLinkPlugin } from "../src/plugins/magic-link";
import { oauthClientPlugin } from "../src/plugins/oauth";
import { mockFetch, type MockReply, type Recorded } from "./helpers";

const userBody = { user: { id: "u1", email: "ada@example.com", email_verified_at: null, first_name: "Ada" } };

function setup(reply: (req: Recorded) => MockReply, redirectFn?: (url: string) => boolean) {
  const { impl, calls } = mockFetch(reply);
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [credentialPasswordPlugin(), magicLinkPlugin(), oauthClientPlugin(), bearerPlugin()],
    fetchOptions: { impl },
    crossTabSync: false,
    refetchOnWindowFocus: false,
    ...(redirectFn ? { redirectFn } : {}),
  });
  return { auth, calls };
}

describe("createAuthClient — session effects", () => {
  it("signIn.credential applies defaults, parses the session, and writes the store", async () => {
    const { auth, calls } = setup(() => ({ body: userBody }));

    const session = await auth.signIn.credential({ credential: "ada@example.com", password: "pw" });

    expect(calls[0]?.body).toEqual({ credential: "ada@example.com", password: "pw", remember_me: true });
    expect((session.user as { firstName?: string }).firstName).toBe("Ada");
    expect(auth.$session.get().data?.user.id).toBe("u1");
  });

  it("signout clears the session store", async () => {
    const { auth } = setup(() => ({ body: userBody }));
    await auth.signIn.credential({ credential: "ada@example.com", password: "pw" });

    await auth.signout();

    expect(auth.$session.get().data).toBeNull();
  });

  it("verifyEmail refetches /me", async () => {
    const { auth, calls } = setup((req) => (req.url.endsWith("/verify-email") ? { body: "ok" } : { body: userBody }));

    await auth.verifyEmail({ token: "tok" });

    expect(calls.map((call) => call.url)).toEqual([
      "http://localhost:8080/auth/verify-email",
      "http://localhost:8080/auth/me",
    ]);
  });

  it("getSession revalidates against /me and returns the store value", async () => {
    const { auth } = setup(() => ({ body: userBody }));

    const session = await auth.getSession();

    expect(session?.user.id).toBe("u1");
    expect(auth.$session.get().data?.user.id).toBe("u1");
  });

  it("does not store a non-session sign-in body", async () => {
    const { auth } = setup(() => ({ body: { two_factor_required: true } }));

    const result = await auth.signIn.credential({ credential: "ada@example.com", password: "pw" });

    expect(result).toEqual({ twoFactorRequired: true });
    expect(auth.$session.get().data).toBeNull();
  });

  it("stores a custom session shape when parseSession returns one", async () => {
    const { impl } = mockFetch(() => ({
      body: { account: { id: "u9", email: "ada@example.com" } },
    }));
    const auth = createAuthClient({
      baseURL: "http://localhost:8080",
      plugins: [credentialPasswordPlugin()],
      fetchOptions: { impl },
      crossTabSync: false,
      refetchOnWindowFocus: false,
      parseSession: (raw) => {
        if (typeof raw !== "object" || raw === null || !("account" in raw)) {
          return false;
        }
        const account = (raw as { account: { id: string; email: string } }).account;
        return { user: { id: account.id, email: account.email, emailVerifiedAt: null } };
      },
    });

    const session = await auth.signIn.credential({ credential: "ada@example.com", password: "pw" });

    expect(session.user.id).toBe("u9");
    expect(auth.$session.get().data?.user.id).toBe("u9");
  });

  it("treats a false parse of /me as signed out", async () => {
    const { impl } = mockFetch(() => ({ body: { status: "anonymous" } }));
    const auth = createAuthClient({
      baseURL: "http://localhost:8080",
      plugins: [credentialPasswordPlugin()],
      fetchOptions: { impl },
      crossTabSync: false,
      refetchOnWindowFocus: false,
      parseSession: () => false,
    });

    await expect(auth.getSession()).resolves.toBeNull();
    expect(auth.$session.get().data).toBeNull();
  });
});

describe("createAuthClient — custom parse / handlers", () => {
  it("username.checkAvailability unwraps `{ available }` to a boolean", async () => {
    const { auth } = setup(() => ({ body: { available: false } }));

    await expect(auth.username.checkAvailability({ username: "ada" })).resolves.toBe(false);
  });

  it("signIn.social redirects unless disableRedirect is set", async () => {
    const redirectFn = vi.fn(() => true);
    const { auth, calls } = setup(() => ({ body: { url: "https://provider/auth" } }), redirectFn);

    const redirected = await auth.signIn.social({ provider: "github" });
    expect(redirectFn).toHaveBeenCalledWith("https://provider/auth");
    expect(redirected.redirect).toBe(true);

    redirectFn.mockClear();
    const skipped = await auth.signIn.social({
      provider: "google",
      redirectUri: "https://app/cb",
      disableRedirect: true,
    });
    expect(redirectFn).not.toHaveBeenCalled();
    expect(skipped).toEqual({ url: "https://provider/auth", redirect: false });
    expect(calls[1]?.url).toBe("http://localhost:8080/auth/oauth/google/authorize?redirect_uri=https%3A%2F%2Fapp%2Fcb");
    expect(calls[1]?.url).not.toContain("provider");
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

    expect(onSuccess).toHaveBeenCalledWith(session);
  });

  it("onError fires with the LimenError and the call still rejects", async () => {
    const { auth } = setup(() => ({ status: 500, body: { message: "boom" } }));
    const onError = vi.fn();

    await expect(auth.sessions(undefined, { onError })).rejects.toBeInstanceOf(LimenError);
    expect(onError.mock.calls[0]?.[0]).toBeInstanceOf(LimenError);
  });
});
