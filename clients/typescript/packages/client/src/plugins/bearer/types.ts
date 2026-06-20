export type BearerPlugin = {
  bearer: {
    /** Currently stored tokens, or `null` if signed out / never set. */
    getTokens: () => BearerTokens | null;
    /** Manually persist tokens. */
    setTokens: (tokens: BearerTokens) => void;
    /** Discard stored tokens. Also runs automatically on `signOut`. */
    clear: () => void;
  };
};

export type BearerTokens = {
  accessToken: string;
  refreshToken?: string;
};

export interface BearerStorage {
  get(): BearerTokens | null;
  set(tokens: BearerTokens): void;
  clear(): void;
}

export type BearerPluginConfig = {
  /**
   * Where to persist tokens. Defaults to `localStorage` when available,
   * otherwise an in-memory store (SSR / non-browser).
   */
  storage?: BearerStorage;
  /**
   * Key used by the default `localStorage` adapter. Ignored when a custom
   * `storage` is supplied. Defaults to `"limen.tokens"`.
   */
  storageKey?: string;
};
