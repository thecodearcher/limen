import { camelToSnake } from "./helpers";

/**
 * Default request serializer: shallow camelCase → snake_case, drops `undefined`,
 * leaves non-objects unchanged.
 *
 * `additionalFields` entries are merged into the top-level body verbatim.
 * Known route fields win on key collisions.
 */
export function defaultSerialize(input: unknown): unknown {
  if (input === null || input === undefined) {
    return input;
  }
  if (typeof input !== "object" || Array.isArray(input)) {
    return input;
  }
  const { additionalFields, ...rest } = input as Record<string, unknown>;
  const out: Record<string, unknown> = {};

  if (additionalFields && typeof additionalFields === "object" && !Array.isArray(additionalFields)) {
    Object.assign(out, additionalFields);
  }

  for (const [key, value] of Object.entries(rest)) {
    if (value === undefined) {
      continue;
    }
    out[camelToSnake(key)] = value;
  }
  return out;
}
