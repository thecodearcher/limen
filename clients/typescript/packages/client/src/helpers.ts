export function stripTrailingSlash(s: string): string {
  return s.endsWith("/") ? s.slice(0, -1) : s;
}

/**
 * Ensure a leading slash while preserving `""` as "no path".
 */
export function ensureLeadingSlash(s: string): string {
  if (s === "" || s === "/") {
    return s;
  }
  return s.startsWith("/") ? s : `/${s}`;
}

/**
 * Normalize a base path to `""` or a leading-slash path without a trailing
 * slash.
 */
export function normalizeBasePath(p: string): string {
  if (p === "" || p === "/") {
    return "";
  }
  const withLeading = p.startsWith("/") ? p : `/${p}`;
  return withLeading.endsWith("/") ? withLeading.slice(0, -1) : withLeading;
}

export function joinURL(baseURL: string, path: string): string {
  return stripTrailingSlash(baseURL) + ensureLeadingSlash(path);
}

export function toCamelCaseKey(key: string): string {
  return key.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase());
}

export function kebabToCamel(key: string): string {
  return key.replace(/-([a-z0-9])/g, (_, c: string) => c.toUpperCase());
}

/**
 * Shallow-convert an object's keys from snake_case to camelCase. Non-object
 * inputs and arrays pass through untouched.
 */
export function camelizeKeys<T = Record<string, unknown>>(raw: unknown): T {
  if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
    return raw as T;
  }
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(raw)) {
    out[toCamelCaseKey(key)] = value;
  }
  return out as T;
}

export function camelizeEach<T = Record<string, unknown>>(raw: unknown): T[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw.map((item) => camelizeKeys<T>(item));
}

/**
 * Convert camelCase to snake_case, leaving already-snake-cased keys unchanged.
 */
export function camelToSnake(key: string): string {
  if (key.includes("_")) {
    return key;
  }
  return key.replace(/[A-Z]/g, (m) => `_${m.toLowerCase()}`);
}
