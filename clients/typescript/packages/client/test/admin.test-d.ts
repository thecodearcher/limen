import { expectTypeOf, test } from "vitest";
import { createAuthClient } from "../src";
import { adminPlugin, type AdminUser } from "../src/plugins/admin";

test("ban and impersonation routes exist unless disabled", () => {
  const full = createAuthClient({ baseURL: "http://localhost:8080", plugins: [adminPlugin()] });
  expectTypeOf(full.admin).toHaveProperty("banUser");
  expectTypeOf(full.admin).toHaveProperty("impersonate");

  const base = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [adminPlugin({ bans: false, impersonation: false })],
  });
  expectTypeOf(base.admin).not.toHaveProperty("banUser");
  expectTypeOf(base.admin).not.toHaveProperty("impersonate");
});

test("TFields types the users in admin responses", () => {
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [adminPlugin<{ impersonation: true }, { plan: string }>({ impersonation: true })],
  });

  expectTypeOf(auth.admin.getUser).returns.resolves.toHaveProperty("plan");
  expectTypeOf(auth.admin.impersonate).returns.resolves.toHaveProperty("user").toHaveProperty("plan");
});

test("the session user carries roles and permissions", () => {
  const auth = createAuthClient({ baseURL: "http://localhost:8080", plugins: [adminPlugin()] });

  expectTypeOf(auth.$session.get().data!.user.roles).toEqualTypeOf<string[]>();
  expectTypeOf(auth.$session.get().data!.user.permissions).toEqualTypeOf<string[]>();
  expectTypeOf(auth.$session.get().data!.session?.impersonatedBy).toEqualTypeOf<AdminUser | undefined>();
});
