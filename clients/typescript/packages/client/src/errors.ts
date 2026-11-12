import { unwrapErrorCode } from "./envelope";

/** A server `code`, or one derived from the HTTP status when the body has none. */
export type LimenErrorCode = string;

/** Use the server's `code` when the body has one; otherwise derive one from the HTTP status. */
export function resolveErrorCode(status: number, body: unknown): LimenErrorCode {
  return unwrapErrorCode(body) ?? deriveErrorCode(status);
}

/** Map HTTP status → typed code. Anything unmapped becomes `"unknown"`. */
// prettier-ignore
export function deriveErrorCode(status: number): LimenErrorCode {
  if (status === 401) {return "unauthorized";}
  if (status === 403) {return "forbidden";}
  if (status === 404) {return "not_found";}
  if (status === 409) {return "conflict";}
  if (status === 422 || status === 400) {return "validation_error";}
  if (status === 429) {return "rate_limited";}
  if (status >= 500 && status < 600) {return "server_error";}
  return "unknown";
}

/**
 * The single error type every SDK call throws on non-2xx. Carries the raw
 * server message, the HTTP status, and the server code. When the body has no
 * code, the code is derived from the status.
 */
export class LimenError extends Error {
  override readonly name = "LimenError";
  readonly status: number;
  readonly code: LimenErrorCode;

  constructor(message: string, status: number, code?: LimenErrorCode) {
    super(message);
    this.status = status;
    this.code = code ?? deriveErrorCode(status);
  }

  is(code: LimenErrorCode): boolean {
    return this.code === code;
  }

  get isUnauthorized(): boolean {
    return this.code === "unauthorized";
  }

  get isRateLimited(): boolean {
    return this.code === "rate_limited";
  }

  get isTimeout(): boolean {
    return this.code === "timeout";
  }
}
