import { defaultSerialize } from "../../serialize";
import type {
  BeginAuthenticationInput,
  BeginRegistrationInput,
  FinishRegistrationInput,
  PasskeyPluginConfig,
} from "./types";

export const MISSING_BROWSER =
  "Passkey ceremonies require @simplewebauthn/browser. Install it with: npm install @simplewebauthn/browser";

export async function loadWebAuthn() {
  try {
    return await import("@simplewebauthn/browser");
  } catch {
    throw new Error(MISSING_BROWSER);
  }
}

export function resolveRpID(config: PasskeyPluginConfig): string | undefined {
  if (config.rpID) {
    return config.rpID;
  }
  const hostname = globalThis.location?.hostname;
  if (hostname) {
    return hostname;
  }
  return undefined;
}

export function serializeBeginQuery(input: BeginRegistrationInput | BeginAuthenticationInput): Record<string, string> {
  const raw = (input ?? {}) as Record<string, unknown>;
  const { extensions, ...rest } = raw;
  const out = defaultSerialize(rest) as Record<string, string>;
  if (Array.isArray(extensions) && extensions.length > 0) {
    out.extensions = extensions.join(",");
  }
  return out;
}

export function serializeFinishRegistration(input: FinishRegistrationInput): Record<string, unknown> {
  const { name, createSession, ...credential } = input;
  return {
    ...credential,
    ...(name !== undefined ? { name } : {}),
    ...(createSession !== undefined ? { create_session: createSession } : {}),
  };
}
