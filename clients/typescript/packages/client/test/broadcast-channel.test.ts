import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createBroadcastChannel } from "../src/broadcast-channel";

class FakeBroadcastChannel {
  static peers = new Map<string, Set<FakeBroadcastChannel>>();
  static reset(): void {
    FakeBroadcastChannel.peers.clear();
  }

  onmessage: ((event: MessageEvent) => void) | null = null;
  private closed = false;

  constructor(public readonly name: string) {
    const set = FakeBroadcastChannel.peers.get(name) ?? new Set();
    set.add(this);
    FakeBroadcastChannel.peers.set(name, set);
  }

  postMessage(data: unknown): void {
    for (const peer of FakeBroadcastChannel.peers.get(this.name) ?? []) {
      if (peer === this || peer.closed) {
        continue;
      }
      peer.onmessage?.({ data: structuredClone(data) } as MessageEvent);
    }
  }

  close(): void {
    this.closed = true;
    FakeBroadcastChannel.peers.get(this.name)?.delete(this);
  }
}

describe("createBroadcastChannel over BroadcastChannel", () => {
  beforeEach(() => {
    FakeBroadcastChannel.reset();
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("delivers a posted message to other ports but not the sender", () => {
    const a = createBroadcastChannel<{ n: number }>("limen.session");
    const b = createBroadcastChannel<{ n: number }>("limen.session");
    const aSeen = vi.fn();
    const bSeen = vi.fn();
    a.subscribe(aSeen);
    b.subscribe(bSeen);

    a.post({ n: 1 });

    expect(bSeen).toHaveBeenCalledWith({ n: 1 });
    expect(aSeen).not.toHaveBeenCalled();

    a.close();
    b.close();
  });

  it("stops delivering to a closed port", () => {
    const a = createBroadcastChannel<{ n: number }>("limen.session");
    const b = createBroadcastChannel<{ n: number }>("limen.session");
    const bSeen = vi.fn();
    b.subscribe(bSeen);
    b.close();

    a.post({ n: 1 });

    expect(bSeen).not.toHaveBeenCalled();
    a.close();
  });

  it("removes a listener when its unsubscribe is called", () => {
    const a = createBroadcastChannel<{ n: number }>("limen.session");
    const b = createBroadcastChannel<{ n: number }>("limen.session");
    const bSeen = vi.fn();
    const off = b.subscribe(bSeen);
    off();

    a.post({ n: 1 });

    expect(bSeen).not.toHaveBeenCalled();
    a.close();
    b.close();
  });
});

describe("createBroadcastChannel storage fallback", () => {
  beforeEach(() => {
    vi.stubGlobal("BroadcastChannel", undefined);
    globalThis.localStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    globalThis.localStorage.clear();
  });

  it("posts via localStorage then clears it", () => {
    const port = createBroadcastChannel<{ n: number }>("limen.session");
    const setItem = vi.spyOn(globalThis.localStorage, "setItem");

    port.post({ n: 1 });

    expect(setItem).toHaveBeenCalledWith("limen.session-sync", JSON.stringify({ n: 1 }));
    expect(globalThis.localStorage.getItem("limen.session-sync")).toBeNull();

    setItem.mockRestore();
    port.close();
  });

  it("emits on a storage event for its key", () => {
    const port = createBroadcastChannel<{ n: number }>("limen.session");
    const seen = vi.fn();
    port.subscribe(seen);

    window.dispatchEvent(
      new StorageEvent("storage", { key: "limen.session-sync", newValue: JSON.stringify({ n: 1 }) }),
    );

    expect(seen).toHaveBeenCalledWith({ n: 1 });
    port.close();
  });

  it("ignores storage events for other keys", () => {
    const port = createBroadcastChannel("limen.session");
    const seen = vi.fn();
    port.subscribe(seen);

    window.dispatchEvent(new StorageEvent("storage", { key: "limen.tokens", newValue: "{}" }));

    expect(seen).not.toHaveBeenCalled();
    port.close();
  });
});

describe("createBroadcastChannel in non-browser environments", () => {
  it("is a no-op when window is undefined", () => {
    vi.stubGlobal("window", undefined);
    try {
      const port = createBroadcastChannel<{ n: number }>("limen.session");
      const seen = vi.fn();
      const off = port.subscribe(seen);

      expect(() => port.post({ n: 1 })).not.toThrow();
      expect(seen).not.toHaveBeenCalled();
      expect(() => off()).not.toThrow();
      expect(() => port.close()).not.toThrow();
    } finally {
      vi.unstubAllGlobals();
    }
  });
});
