import { SET_AUTH_TOKEN_HEADER, SET_REFRESH_TOKEN_HEADER } from "../../constants";
import type { SessionJwtTokens } from "./types";

/**
 * Read the `exp` (seconds since epoch) from a JWT without verifying its
 * signature — the server is the source of truth; the client only needs expiry
 * to decide when to refresh. Returns `null` for anything malformed.
 */
export function decodeJwtExp(token: string): number | null {
  const payload = token.split(".")[1];
  if (!payload) {
    return null;
  }
  try {
    const b64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
    const claims = JSON.parse(atob(padded)) as { exp?: unknown };
    return typeof claims.exp === "number" ? claims.exp : null;
  } catch {
    return null;
  }
}

export function isExpiring(token: string, skewMs: number): boolean {
  const exp = decodeJwtExp(token);
  if (exp === null) {
    return true;
  }
  return exp * 1000 - skewMs <= Date.now();
}

export function tokensFromHeaders(headers: Headers): SessionJwtTokens | null {
  const accessToken = headers.get(SET_AUTH_TOKEN_HEADER);
  if (!accessToken) {
    return null;
  }

  const tokens: SessionJwtTokens = { accessToken };
  const refreshToken = headers.get(SET_REFRESH_TOKEN_HEADER);
  if (refreshToken) {
    tokens.refreshToken = refreshToken;
  }
  return tokens;
}
