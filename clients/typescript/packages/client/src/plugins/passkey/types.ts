import type {
  AuthenticationExtensionsClientInputs,
  AuthenticationResponseJSON,
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
  RegistrationResponseJSON,
} from "@simplewebauthn/browser";
import type { PaginationInput, Session } from "../../types";

export type PasskeyPluginConfig = {
  /**
   * Relying Party ID used for WebAuthn Signal APIs after delete.
   * Defaults to `location.hostname` in the browser.
   */
  rpID?: string;
};

export type Passkey = {
  id: string | number;
  name: string | null;
  credentialId: string;
  transports: string[] | null;
  aaguid: string | null;
  backupEligible: boolean;
  backupState: boolean;
  lastUsedAt: string | null;
  createdAt: string;
  updatedAt: string;
};

export type AuthenticatorAttachment = "platform" | "cross-platform";

export type PasskeyExtensionName = keyof AuthenticationExtensionsClientInputs;

export type BeginPasskeyResult<TPublicKey> = {
  publicKey: TPublicKey;
  mediation?: string;
};

export type BeginRegistrationInput = {
  authenticatorAttachment?: AuthenticatorAttachment;
  /** Opaque signup context (e.g. email) when the server allows public registration. */
  context?: string;
  extensions?: PasskeyExtensionName[];
};

export type BeginRegistrationResult = BeginPasskeyResult<PublicKeyCredentialCreationOptionsJSON>;

export type FinishRegistrationInput = RegistrationResponseJSON & {
  name?: string;
  createSession?: boolean;
};

export type FinishRegistrationResult<TFields = unknown> = Passkey | Session<TFields>;

export type BeginAuthenticationInput = {
  extensions?: PasskeyExtensionName[];
} | void;

export type BeginAuthenticationResult = BeginPasskeyResult<PublicKeyCredentialRequestOptionsJSON>;

export type FinishAuthenticationInput = AuthenticationResponseJSON;

export type ListPasskeysInput = PaginationInput | void;

export type UpdatePasskeyInput = {
  id: string | number;
  name: string;
};

export type DeletePasskeyInput = {
  id: string | number;
  /**
   * When set, best-effort `sendSignal(unknownCredential)` so the password
   * manager may hide or delete the device copy. Use {@link Passkey.credentialId}.
   */
  credentialId?: string;
};

export type RegisterPasskeyInput = {
  name?: string;
  authenticatorAttachment?: AuthenticatorAttachment;
  context?: string;
  createSession?: boolean;
  /** Try silent passkey creation after a non-passkey sign-in. */
  useAutoRegister?: boolean;
  extensions?: PasskeyExtensionName[];
};

export type SignInPasskeyInput = {
  useBrowserAutofill?: boolean;
  verifyBrowserAutofillInput?: boolean;
  extensions?: PasskeyExtensionName[];
} | void;
