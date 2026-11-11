import { createStore, type DataStore } from "../../data-store";
import { storeRef } from "../../stores";
import type { BearerStorage, BearerTokens } from "./types";

export const TOKENS = "tokens";

/** The tokens held by the bearer or session-jwt plugin, whichever is registered. */
export const tokensStore = storeRef<BearerTokens | null>(TOKENS);

/**
 * Publish the tokens in `storage` as a store. Reads go to `storage` every time
 * so a token another tab rotated is never read stale.
 */
export function createTokenStore(storage: BearerStorage): DataStore<BearerTokens | null> {
  const store = createStore<BearerTokens | null>({ initial: storage.get(), settled: true });
  return {
    $state: store.$state,
    current: () => ({ ...store.current(), data: storage.get() }),
    setData: (tokens) => {
      if (tokens === null) {
        storage.clear();
      } else {
        storage.set(tokens);
      }
      store.setData(tokens);
    },
    refetch: store.refetch,
  };
}
