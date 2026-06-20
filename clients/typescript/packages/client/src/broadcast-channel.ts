/**
 * Cross-tab message transport: callers post and
 * subscribe to typed messages. Uses `BroadcastChannel` where available, falling
 * back to `localStorage` `storage` events. A no-op in non-browser environments.
 */
type BroadcastPort<T> = {
  post(message: T): void;
  subscribe(listener: (message: T) => void): () => void;
  close(): void;
};

const NOOP_PORT: BroadcastPort<unknown> = {
  post() {},
  subscribe: () => () => {},
  close() {},
};

export function createBroadcastChannel<T>(name: string): BroadcastPort<T> {
  if (typeof window === "undefined") {
    return NOOP_PORT as BroadcastPort<T>;
  }

  const storageKey = `${name}-sync`;
  const listeners = new Set<(message: T) => void>();
  const emit = (message: T): void => {
    for (const listener of listeners) {
      listener(message);
    }
  };

  const channel = typeof BroadcastChannel !== "undefined" ? new BroadcastChannel(name) : null;
  if (channel) {
    channel.onmessage = (event: MessageEvent<T>) => {
      if (event.data != null) {
        emit(event.data);
      }
    };
  }

  const onStorage = (event: StorageEvent): void => {
    if (event.key !== storageKey || !event.newValue) {
      return;
    }

    emit(JSON.parse(event.newValue) as T);
  };

  if (!channel) {
    window.addEventListener("storage", onStorage);
  }

  return {
    post(message) {
      if (channel) {
        channel.postMessage(message);
        return;
      }

      if (typeof globalThis.localStorage !== "undefined") {
        globalThis.localStorage.setItem(storageKey, JSON.stringify(message));
        globalThis.localStorage.removeItem(storageKey);
      }
    },
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    close() {
      listeners.clear();
      if (channel) {
        channel.close();
        return;
      }
      window.removeEventListener("storage", onStorage);
    },
  };
}
