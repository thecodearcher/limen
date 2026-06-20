import { toCamelCaseKey } from "./helpers";
import type { Session, User } from "./types";

/**
 * Convert snake_case User keys from server payloads to camelCase. Unknown
 * extension fields are converted with the same rule so consumers consistently
 * read camelCase in SDK responses.
 */
export function normalizeUser<F = unknown>(raw: Record<string, unknown>): User<F> {
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(raw)) {
    out[toCamelCaseKey(key)] = value;
  }
  return out as User<F>;
}

export function defaultSessionParse<F = unknown>(raw: unknown): Session<F> {
  if (!raw || typeof raw !== "object") {
    throw new TypeError(`Expected session response to be an object, got ${raw === null ? "null" : typeof raw}`);
  }
  const obj = raw as Record<string, unknown>;
  const userRaw = (obj["user"] ?? obj) as Record<string, unknown>;
  return { user: normalizeUser<F>(userRaw) };
}
