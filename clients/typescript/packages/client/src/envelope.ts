import { DEFAULT_ENVELOPE_FIELDS } from "./constants";
import { type EnvelopeConfig } from "./types";

/**
 * Unwrap a parsed JSON success body according to envelope config.
 *
 * - `mode: "off"`: return as-is.
 * - `mode: "wrap-success" | "always"`: extract `body[fields.data]` if present;
 *   if the key is missing (server didn't wrap this particular response), fall
 *   back to the raw body so we don't lose data.
 *
 * Returning `unknown` is intentional: the caller knows the expected shape and
 * narrows / validates it. Trying to be clever here would invite false typing.
 */
export function unwrapPayload(body: unknown, envelope: EnvelopeConfig): unknown {
  if (envelope.mode === "off") {
    return body;
  }
  if (body === null || typeof body !== "object") {
    return body;
  }
  const fields = envelope.fields ?? DEFAULT_ENVELOPE_FIELDS;
  const record = body as Record<string, unknown>;
  if (fields.data in record) {
    return record[fields.data];
  }
  return body;
}

/**
 * Pull the human-readable error message out of a non-2xx body.
 *
 * - `mode: "off" | "wrap-success"`: error bodies look like `{ message }`
 * - `mode: "always"`: error bodies use the configured `fields.message` key.
 *
 * Returns `undefined` when no message can be located; the caller substitutes
 * a status-based fallback (e.g. HTTP status text).
 */
export function unwrapErrorMessage(body: unknown, envelope: EnvelopeConfig): string | undefined {
  if (body === null || typeof body !== "object") {
    return undefined;
  }
  const record = body as Record<string, unknown>;
  const fields = envelope.fields ?? DEFAULT_ENVELOPE_FIELDS;
  const key = envelope.mode === "always" ? fields.message : "message";
  const value = record[key];
  return typeof value === "string" ? value : undefined;
}
