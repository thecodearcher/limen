import type { BearerPluginConfig, BearerTokens } from "../bearer";

export type SessionJwtTokens = BearerTokens;

export type SessionJwtPluginConfig = BearerPluginConfig & {
  /** Refresh once the access token is within this many seconds of expiry. Default 30. */
  expirySkewSeconds?: number;
};

export type RefreshInput = { refreshToken: string };
