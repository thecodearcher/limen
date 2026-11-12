import { DEFAULT_TOKEN_STORAGE_KEY, SET_AUTH_TOKEN_HEADER, SET_REFRESH_TOKEN_HEADER } from "../../constants";
import { defineClientPlugin, defineRoutes, defineStores } from "../../define-plugin";
import { resolveDefaultStorage } from "./storage";
import { createTokenStore, TOKENS } from "./token-store";
import type { BearerPluginConfig, BearerTokens } from "./types";

export function bearerPlugin(config: BearerPluginConfig = {}) {
  const store = config.storage ?? resolveDefaultStorage(config.storageKey ?? DEFAULT_TOKEN_STORAGE_KEY);
  const tokenStore = createTokenStore(store);

  return defineClientPlugin({
    id: "bearer",
    routes: defineRoutes(),
    stores: () => defineStores({ [TOKENS]: tokenStore }),
    actions: () => ({
      bearer: {
        getTokens: () => store.get(),
        setTokens: (tokens: BearerTokens) => tokenStore.setData(tokens),
        clear: () => tokenStore.setData(null),
      },
    }),
    hooks: {
      beforeRequest: [
        {
          run: (req) => {
            const tokens = store.get();
            if (tokens?.accessToken && !req.headers.has("Authorization")) {
              req.headers.set("Authorization", `Bearer ${tokens.accessToken}`);
            }
            return req;
          },
        },
      ],
      afterResponse: [
        {
          allowOnFailure: true,
          run: (res) => {
            const accessToken = res.headers.get(SET_AUTH_TOKEN_HEADER);
            if (accessToken) {
              const current = store.get();
              const tokens: BearerTokens = { accessToken };
              const refreshToken = res.headers.get(SET_REFRESH_TOKEN_HEADER) ?? current?.refreshToken;
              if (refreshToken) {
                tokens.refreshToken = refreshToken;
              }
              tokenStore.setData(tokens);
            }
            return res;
          },
        },
        {
          match: ["/signout", "/revoke-sessions"],
          run: (res) => {
            tokenStore.setData(null);
            return res;
          },
        },
      ],
    },
  });
}

export { localStorageBearerStorage, memoryBearerStorage, resolveDefaultStorage } from "./storage";
export { createTokenStore, tokensStore } from "./token-store";
export type { BearerPluginConfig, BearerPlugin as BearerPublic, BearerStorage, BearerTokens } from "./types";
