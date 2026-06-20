import { DEFAULT_TOKEN_STORAGE_KEY } from "../../constants";
import type { AnyRouteContext } from "../../context";
import { defineClientPlugin, defineRoutes, type RunRoute } from "../../define-plugin";
import { LimenError } from "../../errors";
import { route } from "../../route";
import type { Session } from "../../types";
import { resolveDefaultStorage } from "../bearer";
import { DEFAULT_EXPIRY_SKEW_SECONDS } from "./constants";
import { isExpiring, tokensFromHeaders } from "./jwt";
import type { RefreshInput, SessionJwtPluginConfig, SessionJwtTokens } from "./types";

export function sessionJwtPlugin<TFields = unknown>(config: SessionJwtPluginConfig = {}) {
  const store = config.storage ?? resolveDefaultStorage(config.storageKey ?? DEFAULT_TOKEN_STORAGE_KEY);
  const skewMs = (config.expirySkewSeconds ?? DEFAULT_EXPIRY_SKEW_SECONDS) * 1000;

  const refreshRoute = route<RefreshInput, Session<TFields>>()({
    method: "POST",
    path: "/refresh",
    parseSession: true,
    expose: false,
  });

  let ctx!: AnyRouteContext;
  let run!: RunRoute;
  let inFlight: Promise<SessionJwtTokens> | null = null;

  const runRefresh = async (): Promise<SessionJwtTokens> => {
    const current = store.get();
    if (!current?.refreshToken) {
      throw new LimenError("No refresh token found", 401, "unauthorized");
    }
    try {
      await run(refreshRoute, { refreshToken: current.refreshToken });
      const tokens = store.get();
      if (!tokens?.accessToken) {
        throw new LimenError("Refresh did not return a valid access token", 500, "unknown");
      }
      return tokens;
    } catch (err) {
      store.clear();
      ctx.setSession(null);
      throw err;
    }
  };

  const refresh = (): Promise<SessionJwtTokens> => {
    if (!inFlight) {
      inFlight = runRefresh().finally(() => {
        inFlight = null;
      });
    }
    return inFlight;
  };

  const getAccessToken = async (): Promise<string | null> => {
    const current = store.get();
    if (!current?.accessToken) {
      return null;
    }

    if (!isExpiring(current.accessToken, skewMs)) {
      return current.accessToken;
    }

    if (!current.refreshToken) {
      return null;
    }
    return (await refresh()).accessToken;
  };

  const applyAuthHeader = async (req: { headers: Headers }): Promise<void> => {
    if (req.headers.has("Authorization")) {
      return;
    }
    const token = await getAccessToken().catch(() => null);
    if (token) {
      req.headers.set("Authorization", `Bearer ${token}`);
    }
  };

  return defineClientPlugin({
    id: "session-jwt",
    routes: defineRoutes(refreshRoute),
    actions: (pluginCtx, pluginRun) => {
      ctx = pluginCtx;
      run = pluginRun;
      return {
        sessionJwt: {
          /**
           * Get the current access token. If the token is expiring, refresh it.
           * @returns The current access token or null if no token is found.
           */
          getAccessToken,
          refresh,
          getTokens: () => store.get(),
          clear: () => store.clear(),
        },
      };
    },
    hooks: {
      beforeRequest: [
        {
          match: (route) => route.path !== "/refresh",
          run: async (req) => {
            await applyAuthHeader(req);
            return req;
          },
        },
      ],
      afterResponse: [
        {
          allowOnFailure: true,
          run: (res) => {
            const tokens = tokensFromHeaders(res.headers);
            if (tokens) {
              store.set(tokens);
            }
            return res;
          },
        },
        {
          match: ["/signout", "/revoke-sessions"],
          allowOnFailure: true,
          run: (res) => {
            store.clear();
            return res;
          },
        },
      ],
    },
  });
}

export type { SessionJwtPluginConfig, SessionJwtTokens } from "./types";
