import { DEFAULT_TOKEN_STORAGE_KEY, SET_AUTH_TOKEN_HEADER, SET_REFRESH_TOKEN_HEADER } from "../../constants";
import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { resolveDefaultStorage } from "./storage";
import type { BearerPluginConfig, BearerTokens } from "./types";

export function bearerPlugin(config: BearerPluginConfig = {}) {
  const store = config.storage ?? resolveDefaultStorage(config.storageKey ?? DEFAULT_TOKEN_STORAGE_KEY);

  return defineClientPlugin({
    id: "bearer",
    routes: defineRoutes(),
    actions: () => ({
      bearer: {
        getTokens: () => store.get(),
        setTokens: (tokens: BearerTokens) => store.set(tokens),
        clear: () => store.clear(),
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
              const tokens: BearerTokens = { accessToken };
              const refreshToken = res.headers.get(SET_REFRESH_TOKEN_HEADER);
              if (refreshToken) {
                tokens.refreshToken = refreshToken;
              }
              store.set(tokens);
            }
            return res;
          },
        },
        {
          match: ["/signout", "/revoke-sessions"],
          run: (res) => {
            store.clear();
            return res;
          },
        },
      ],
    },
  });
}

export { localStorageBearerStorage, memoryBearerStorage, resolveDefaultStorage } from "./storage";
export type { BearerPluginConfig, BearerPlugin as BearerPublic, BearerStorage, BearerTokens } from "./types";
