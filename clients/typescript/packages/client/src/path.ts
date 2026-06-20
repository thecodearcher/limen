import { kebabToCamel } from "./helpers";

/**
 * Derive the public client chain from a route path. Mirrors the type-level
 * `PathSegments` in `infer.ts` exactly, so runtime materialization and compile
 * time inference can never drift.
 *
 *   "/otp/send"            -> ["otp", "send"]
 *   "/revoke-sessions"     -> ["revokeSessions"]
 *   "/:provider/authorize" -> ["authorize"]   (param segments are dropped)
 */
export function pathToChain(path: string): string[] {
  return path
    .split("/")
    .filter((seg) => seg.length > 0 && !seg.startsWith(":"))
    .map(kebabToCamel);
}

export function chainFromDotted(chain: string): string[] {
  return chain.split(".").filter((seg) => seg.length > 0);
}

type ResolvedPath = {
  /** Path with `:param` segments substituted. */
  path: string;
  /** Input with declared param keys removed. */
  rest: unknown;
};

/**
 * Substitute declared path params from `input` into `path` and strip them from
 * the payload.
 */
export function resolvePath(path: string, params: readonly string[] | undefined, input: unknown): ResolvedPath {
  if (!params || params.length === 0) {
    return { path, rest: input };
  }
  const source = (input ?? {}) as Record<string, unknown>;
  const rest: Record<string, unknown> = { ...source };
  let resolved = path;
  for (const param of params) {
    const value = rest[param];
    if (value === undefined) {
      throw new Error(`Missing required path param "${param}" for route "${path}"`);
    }
    resolved = resolved.replace(`:${param}`, encodeURIComponent(value as string));
    delete rest[param];
  }
  return { path: resolved, rest };
}
