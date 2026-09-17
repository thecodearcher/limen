import type { StrictConfig } from "../../define-plugin";
import { defineClientPlugin } from "../../define-plugin";
import { buildPasskeyActions } from "./actions";
import { buildPasskeyRoutes } from "./routes";
import type { PasskeyPluginConfig } from "./types";

/**
 * Passkey / WebAuthn client plugin. Orchestrates registration and sign-in
 * ceremonies via `@simplewebauthn/browser` (optional peer dependency).
 *
 * @example
 *   import { passkey } from "limen-auth/plugins/passkey";
 *   createAuthClient({ plugins: [passkey({ rpID: "example.com" })] });
 */
export function passkey<
  const Config extends StrictConfig<Config, PasskeyPluginConfig> = Record<never, never>,
  TFields = unknown,
>(config?: Config) {
  const resolved: PasskeyPluginConfig = config ?? {};
  const routeSet = buildPasskeyRoutes<TFields>();

  return defineClientPlugin({
    id: "passkey",
    basePath: "/passkeys",
    routes: routeSet.routes,
    actions: (_ctx, run) => buildPasskeyActions(resolved, routeSet, run),
  });
}

export type {
  AuthenticatorAttachment,
  BeginAuthenticationInput,
  BeginAuthenticationResult,
  BeginPasskeyResult,
  BeginRegistrationInput,
  BeginRegistrationResult,
  DeletePasskeyInput,
  FinishAuthenticationInput,
  FinishRegistrationInput,
  FinishRegistrationResult,
  ListPasskeysInput,
  Passkey,
  PasskeyExtensionName,
  PasskeyPluginConfig,
  RegisterPasskeyInput,
  SignInPasskeyInput,
  UpdatePasskeyInput,
} from "./types";
