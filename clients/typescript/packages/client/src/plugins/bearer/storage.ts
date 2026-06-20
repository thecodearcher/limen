import { DEFAULT_TOKEN_STORAGE_KEY } from "../../constants";
import type { BearerStorage, BearerTokens } from "./types";

export function memoryBearerStorage(): BearerStorage {
  let tokens: BearerTokens | null = null;
  return {
    get: () => tokens,
    set: (next) => {
      tokens = next;
    },
    clear: () => {
      tokens = null;
    },
  };
}

export function localStorageBearerStorage(key: string = DEFAULT_TOKEN_STORAGE_KEY): BearerStorage {
  return {
    get: () => {
      try {
        const raw = globalThis.localStorage.getItem(key);
        if (!raw) {
          return null;
        }
        const parsed = JSON.parse(raw) as Partial<BearerTokens>;
        if (typeof parsed?.accessToken !== "string") {
          return null;
        }
        return parsed as BearerTokens;
      } catch {
        return null;
      }
    },
    set: (tokens) => {
      globalThis.localStorage.setItem(key, JSON.stringify(tokens));
    },
    clear: () => {
      globalThis.localStorage.removeItem(key);
    },
  };
}

export function resolveDefaultStorage(key: string): BearerStorage {
  if (typeof globalThis.localStorage !== "undefined") {
    return localStorageBearerStorage(key);
  }
  return memoryBearerStorage();
}
