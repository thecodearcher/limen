export type LimenErrorCode =
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "rate_limited"
  | "validation_error"
  | "conflict"
  | "server_error"
  | "unknown";

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
 * server message, the HTTP status, and a derived typed code.
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

  get isUnauthorized(): boolean {
    return this.code === "unauthorized";
  }

  get isRateLimited(): boolean {
    return this.code === "rate_limited";
  }
}
