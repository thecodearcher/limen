import { describe, expect, it, vi } from "vitest";
import { createAuthClient } from "../src";
import { createStore } from "../src/data-store";
import { defineClientPlugin, defineRoutes, defineStores } from "../src/define-plugin";
import type { SessionState } from "../src/session-store";
import { sessionStore } from "../src/session-store";
import { effect, storeRef } from "../src/stores";
import type { PrettySessionFields, PrettyUserFields, Session } from "../src/types";
import { mockFetch, settle, type MockReply, type Recorded } from "./helpers";

const signedIn: Session = { user: { id: "u1", email: "ada@example.com", emailVerifiedAt: null } };

function setup<const Plugins extends Parameters<typeof createAuthClient>[0]["plugins"] & readonly unknown[]>(
  plugins: Plugins,
  reply: (req: Recorded) => MockReply = () => ({ status: 204 }),
) {
  const { impl, calls } = mockFetch(reply);
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins,
    fetchOptions: { impl },
    initialSession: signedIn as Session<PrettyUserFields<Plugins, unknown>, PrettySessionFields<Plugins>>,
    crossTabSync: false,
    refetchOnWindowFocus: false,
  });
  return { auth, calls };
}

describe("store registry", () => {
  it("exposes plugin stores on the client under their `$` names", () => {
    const probe = defineClientPlugin({
      id: "probe",
      routes: defineRoutes(),
      stores: () => defineStores({ probe: createStore<string | null>({ initial: null }) }),
    });
    const { auth } = setup([probe]);

    expect(auth.$probe).toBeDefined();
    expect(auth.$session).toBeDefined();
  });

  it("rejects a duplicate store name", () => {
    const first = defineClientPlugin({
      id: "first",
      routes: defineRoutes(),
      stores: () => defineStores({ probe: createStore<string | null>({ initial: null }) }),
    });
    const clashing = defineClientPlugin({
      id: "clashing",
      routes: defineRoutes(),
      stores: () => defineStores({ probe: createStore<string | null>({ initial: null }) }),
    });

    expect(() => setup([first, clashing])).toThrow(/probe/);
  });
});

describe("store effects", () => {
  it("observes session writes without mounting the session store", async () => {
    const seen: Array<SessionState["data"]> = [];
    const recorder = defineClientPlugin({
      id: "recorder",
      routes: defineRoutes(),
      effects: [
        effect(sessionStore, (state) => {
          seen.push(state.data);
        }),
      ],
    });
    const { auth, calls } = setup([recorder]);

    await auth.signout();

    expect(seen).toEqual([null]);
    // Mounting the session store would hydrate it with `GET /me`.
    expect(calls.map((call) => call.url)).toEqual(["http://localhost:8080/auth/signout"]);
  });

  it("reports a failing effect without breaking the write that triggered it", async () => {
    const errors = vi.spyOn(console, "error").mockImplementation(() => {});
    const failing = defineClientPlugin({
      id: "failing",
      routes: defineRoutes(),
      effects: [
        effect(sessionStore, () => {
          throw new Error("effect blew up");
        }),
      ],
    });
    const seen: Array<SessionState["data"]> = [];
    const recorder = defineClientPlugin({
      id: "recorder",
      routes: defineRoutes(),
      effects: [
        effect(sessionStore, (state) => {
          seen.push(state.data);
        }),
      ],
    });
    const { auth } = setup([failing, recorder]);

    await expect(auth.signout()).resolves.toBeUndefined();

    expect(errors).toHaveBeenCalledOnce();
    expect(seen).toEqual([null]);
    errors.mockRestore();
  });

  it("runs when:data effects only when data changes, not on pending", async () => {
    const seen: Array<string | null> = [];
    const probeStore = storeRef<string | null>("probe");
    let resolveLoad!: (value: string) => void;

    const probe = defineClientPlugin({
      id: "probe",
      routes: defineRoutes(),
      stores: () =>
        defineStores({
          probe: createStore<string | null>({
            initial: null,
            loader: () =>
              new Promise<string>((resolve) => {
                resolveLoad = resolve;
              }),
          }),
        }),
      effects: [
        effect(probeStore, (state) => {
          seen.push(state.data);
        }),
      ],
      actions: (ctx) => ({
        probe: {
          load: () => ctx.stores.get(probeStore)?.refetch() ?? Promise.resolve(),
        },
      }),
    });

    const { auth } = setup([probe]);

    const loading = auth.probe.load();
    await settle();
    expect(seen).toEqual([]);

    resolveLoad("ready");
    await loading;

    expect(seen).toEqual(["ready"]);
  });
});

describe("cross-plugin stores", () => {
  const published = storeRef<string | null>("published");

  function publisherPlugin() {
    return defineClientPlugin({
      id: "publisher",
      routes: defineRoutes(),
      stores: () => defineStores({ published: createStore<string | null>({ initial: null, settled: true }) }),
      actions: (ctx) => ({
        publisher: {
          set: (value: string | null) => ctx.stores.get(published)?.setData(value),
        },
      }),
    });
  }

  it("lets another plugin observe and read a published store", async () => {
    const seen: Array<string | null> = [];
    let sessionFromCtx: unknown = "unset";
    const consumer = defineClientPlugin({
      id: "consumer",
      routes: defineRoutes(),
      effects: [
        effect(published, (state, ctx) => {
          seen.push(state.data);
          sessionFromCtx = ctx.currentSession();
        }),
      ],
      actions: (ctx) => ({
        consumer: {
          value: (): string | null => ctx.stores.get(published)?.current().data ?? null,
        },
      }),
    });
    // consumer before publisher — registration order must not matter.
    const { auth } = setup([consumer, publisherPlugin()]);

    expect(auth.consumer.value()).toBeNull();

    auth.publisher.set("acme");
    await settle();

    expect(seen).toEqual(["acme"]);
    expect(auth.consumer.value()).toBe("acme");
    expect(sessionFromCtx).toEqual(signedIn);
  });

  it("skips an effect whose store belongs to an unregistered plugin", async () => {
    const seen: Array<string | null> = [];
    const orphan = defineClientPlugin({
      id: "orphan",
      routes: defineRoutes(),
      effects: [
        effect(published, (state) => {
          seen.push(state.data);
        }),
      ],
    });
    const { auth } = setup([orphan]);

    await expect(auth.signout()).resolves.toBeUndefined();

    expect(seen).toEqual([]);
  });
});
