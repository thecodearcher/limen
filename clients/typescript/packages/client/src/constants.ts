import type { EnvelopeConfig, EnvelopeFields } from "./types";

export const DEFAULT_ENVELOPE_CONFIG: EnvelopeConfig = {
  mode: "off",
};

export const DEFAULT_ENVELOPE_FIELDS: EnvelopeFields = {
  data: "data",
  message: "message",
};

/** Response headers the server emits for the JWT and rotated refresh token. */
export const SET_AUTH_TOKEN_HEADER = "Set-Auth-Token";
export const SET_REFRESH_TOKEN_HEADER = "Set-Refresh-Token";

/** Default `localStorage` key for the persisted tokens. */
export const DEFAULT_TOKEN_STORAGE_KEY = "limen.tokens";
