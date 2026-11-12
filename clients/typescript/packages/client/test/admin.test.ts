import { describe, expect, it } from "vitest";
import { createAuthClient } from "../src";
import { adminPlugin } from "../src/plugins/admin";
import { bearerPlugin, createTokenStore, memoryBearerStorage, type BearerTokens } from "../src/plugins/bearer";
import { sessionJwtPlugin } from "../src/plugins/session-jwt";
import { mockFetch, type MockReply, type Recorded } from "./helpers";

function jwt(sub: string): string {
  const payload = btoa(JSON.stringify({ sub, exp: Math.floor(Date.now() / 1000) + 3600 }));
  return `header.${payload.replace(/=+$/, "")}.signature`;
}

const admin = { id: "admin-1", email: "admin@example.com", emailVerifiedAt: null, roles: ["admin"], permissions: [] };
const target = { id: "user-2", email: "user@example.com", emailVerifiedAt: null };
const adminTokens: BearerTokens = { accessToken: jwt("admin-1"), refreshToken: "admin-refresh" };
const impersonationTokens = { "Set-Auth-Token": jwt("user-2"), "Set-Refresh-Token": "impersonation-refresh" };

const tokenPlugins = [
  { name: "bearer", plugin: bearerPlugin },
  { name: "session-jwt", plugin: sessionJwtPlugin },
] as const;

function setup(tokenPlugin: (typeof tokenPlugins)[number]["plugin"], stop: MockReply = { status: 204 }) {
  const tokens = memoryBearerStorage();
  tokens.set(adminTokens);
  const stash = memoryBearerStorage();
  const { impl, calls } = mockFetch((req: Recorded): MockReply => {
    if (req.url.endsWith("/impersonate")) {
      return { body: { user: target, session: { impersonated_by: admin } }, headers: impersonationTokens };
    }
    if (req.url.endsWith("/impersonation/stop")) {
      return stop;
    }
    if (req.url.endsWith("/signout")) {
      return { status: 204 };
    }
    return { body: { user: admin } };
  });
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [tokenPlugin({ storage: tokens }), adminPlugin({ impersonation: true, storage: stash })],
    fetchOptions: { impl },
    initialSession: { user: admin },
    crossTabSync: false,
    refetchOnWindowFocus: false,
  });
  return { auth, calls, tokens, stash };
}

describe("admin impersonation", () => {
  it.each(tokenPlugins)("restores the admin's tokens after stopping with $name", async ({ plugin }) => {
    const { auth, calls, tokens } = setup(plugin);

    await auth.admin.impersonate({ id: "user-2" });
    expect(tokens.get()?.refreshToken).toBe("impersonation-refresh");
    expect(auth.$session.get().data?.user.id).toBe("user-2");
    expect(auth.$session.get().data?.session?.impersonatedBy?.id).toBe("admin-1");

    await auth.admin.stopImpersonating();

    const stopCall = calls.find((call) => call.url.endsWith("/impersonation/stop"));
    expect(stopCall?.headers.get("Authorization")).toBe(`Bearer ${impersonationTokens["Set-Auth-Token"]}`);
    expect(tokens.get()).toEqual(adminTokens);
    expect(auth.$session.get().data?.user.id).toBe("admin-1");
    expect(auth.$session.get().data?.session?.impersonatedBy).toBeUndefined();
  });

  it("restores the admin's tokens when stopping fails", async () => {
    const { auth, tokens } = setup(bearerPlugin, { status: 401, body: { message: "unauthorized" } });

    await auth.admin.impersonate({ id: "user-2" });

    await expect(auth.admin.stopImpersonating()).rejects.toThrow();
    expect(tokens.get()).toEqual(adminTokens);
  });

  it("drops the admin's tokens when the impersonated session signs out", async () => {
    const { auth, stash } = setup(bearerPlugin);

    await auth.admin.impersonate({ id: "user-2" });
    expect(stash.get()).toEqual(adminTokens);

    await auth.signout();

    expect(stash.get()).toBeNull();
  });
});

describe("token store", () => {
  it("reads tokens written to storage outside the store", () => {
    const storage = memoryBearerStorage();
    const store = createTokenStore(storage);

    storage.set(adminTokens);

    expect(store.current().data).toEqual(adminTokens);
  });
});
