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

/**
 * Map a default Limen session body into a `Session`. Returns `false` unless
 * `user` is a non-null object, so non-session payloads (a pending two-factor
 * challenge, a plain message) are not stored.
 */
export function defaultSessionParse<F = unknown>(raw: unknown): Session<F> | false {
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    return false;
  }
  const userRaw = (raw as Record<string, unknown>)["user"];
  if (typeof userRaw !== "object" || userRaw === null || Array.isArray(userRaw)) {
    return false;
  }
  const user = normalizeUser<F>(userRaw as Record<string, unknown>);
  const sessionRaw = (raw as Record<string, unknown>)["session"];
  if (typeof sessionRaw !== "object" || sessionRaw === null || Array.isArray(sessionRaw)) {
    return { user };
  }
  return { user, session: camelizeDeep(sessionRaw) as Record<never, never> };
}

function camelizeDeep(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(camelizeDeep);
  }
  if (typeof value !== "object" || value === null) {
    return value;
  }
  const out: Record<string, unknown> = {};
  for (const [key, inner] of Object.entries(value)) {
    out[toCamelCaseKey(key)] = camelizeDeep(inner);
  }
  return out;
}
