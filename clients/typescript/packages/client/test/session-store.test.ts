import { describe, expect, it, vi } from "vitest";
import { LimenError } from "../src/errors";
import { createSessionStore } from "../src/session-store";
import type { Session } from "../src/types";

function user(id: string): Session {
  return { user: { id, email: `${id}@example.com`, emailVerifiedAt: null } };
}

function tick(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

describe("createSessionStore lifecycle", () => {
  it("hydrates from the hydrator on the first subscriber", async () => {
    const hydrator = vi.fn(async () => user("user-1"));
    const store = createSessionStore({ hydrator });

    expect(hydrator).not.toHaveBeenCalled();
    const unsubscribe = store.$session.listen(() => {});
    await tick();

    expect(hydrator).toHaveBeenCalledTimes(1);
    expect(store.$session.get().data?.user.id).toBe("user-1");
    unsubscribe();
  });

  it("does not hydrate when seeded with initialSession", async () => {
    const hydrator = vi.fn(async () => user("user-1"));
    const store = createSessionStore({ hydrator, initialSession: user("seed") });

    const unsubscribe = store.$session.listen(() => {});
    await tick();

    expect(hydrator).not.toHaveBeenCalled();
    expect(store.$session.get().data?.user.id).toBe("seed");
    unsubscribe();
  });
});

describe("createSessionStore state", () => {
  it("setData writes the session and clears pending/error", () => {
    // Seed so reading via get() does not trigger onMount hydration.
    const store = createSessionStore({ hydrator: async () => user("user-1"), initialSession: null });

    store.setData(user("user-1"));

    expect(store.$session.get()).toEqual({ data: user("user-1"), isPending: false, error: null });
  });

  it("refetch loads the session from the hydrator", async () => {
    const store = createSessionStore({ hydrator: async () => user("user-1"), initialSession: null });

    await store.refetch();

    expect(store.$session.get().data?.user.id).toBe("user-1");
  });

  it("refetch treats a 401 as signed out", async () => {
    const store = createSessionStore({
      hydrator: async () => {
        throw new LimenError("nope", 401);
      },
      initialSession: user("user-1"),
    });

    await store.refetch();

    expect(store.$session.get().data).toBeNull();
    expect(store.$session.get().error).toBeNull();
  });

  it("refetch preserves the session and surfaces other failures", async () => {
    const store = createSessionStore({
      hydrator: async () => {
        throw new LimenError("boom", 500);
      },
      initialSession: user("user-1"),
    });

    await store.refetch();

    expect(store.$session.get().data?.user.id).toBe("user-1");
    expect(store.$session.get().error?.status).toBe(500);
  });

  it("refetch with maxAgeMs skips when a hydration ran within the window", async () => {
    vi.useFakeTimers();
    try {
      const hydrator = vi.fn(async () => user("user-1"));
      const store = createSessionStore({ hydrator, initialSession: null });

      await store.refetch();
      await store.refetch({ maxAgeMs: 5_000 });

      expect(hydrator).toHaveBeenCalledTimes(1);

      vi.advanceTimersByTime(5_001);
      await store.refetch({ maxAgeMs: 5_000 });

      expect(hydrator).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it("refetch without maxAgeMs always fetches", async () => {
    const hydrator = vi.fn(async () => user("user-1"));
    const store = createSessionStore({ hydrator, initialSession: null });

    await store.refetch();
    await store.refetch();

    expect(hydrator).toHaveBeenCalledTimes(2);
  });

  it("refetch with skipSignedOut does not fetch while signed out", async () => {
    const hydrator = vi.fn(async () => user("user-1"));
    const store = createSessionStore({ hydrator, initialSession: null });

    await store.refetch({ skipSignedOut: true });

    expect(hydrator).not.toHaveBeenCalled();
  });

  it("refetch with skipSignedOut fetches when a session is present", async () => {
    const hydrator = vi.fn(async () => user("user-1"));
    const store = createSessionStore({ hydrator, initialSession: user("seed") });

    await store.refetch({ skipSignedOut: true });

    expect(hydrator).toHaveBeenCalledTimes(1);
  });
});
