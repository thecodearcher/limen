import type { RunRoute } from "../../define-plugin";
import { loadWebAuthn, resolveRpID } from "./helpers";
import type { PasskeyRouteSet } from "./routes";
import type { DeletePasskeyInput, PasskeyPluginConfig, RegisterPasskeyInput, SignInPasskeyInput } from "./types";

export function buildPasskeyActions<TFields>(
  config: PasskeyPluginConfig,
  {
    beginRegistration,
    finishRegistration,
    beginAuthentication,
    finishAuthentication,
    deletePasskey,
  }: PasskeyRouteSet<TFields>,
  run: RunRoute,
) {
  return {
    signIn: {
      passkey: async (input?: SignInPasskeyInput) => {
        const { extensions, ...opts } = (input ?? {}) as Exclude<SignInPasskeyInput, void>;
        const swa = await loadWebAuthn();
        if (!swa.browserSupportsWebAuthn()) {
          throw new Error("This browser does not support passkeys.");
        }
        const began = await run(beginAuthentication, extensions !== undefined ? { extensions } : undefined);
        const assertion = await swa.startAuthentication({
          optionsJSON: began.publicKey,
          ...opts,
        });
        return run(finishAuthentication, assertion);
      },
    },
    passkey: {
      register: async (input?: RegisterPasskeyInput) => {
        const { name, createSession, useAutoRegister, ...beginInput } = input ?? {};
        const swa = await loadWebAuthn();
        if (!swa.browserSupportsWebAuthn()) {
          throw new Error("This browser does not support passkeys.");
        }
        const began = await run(beginRegistration, beginInput);
        const attestation = await swa.startRegistration({
          optionsJSON: began.publicKey,
          useAutoRegister: useAutoRegister ?? false,
        });
        return run(finishRegistration, {
          ...attestation,
          ...(name !== undefined ? { name } : {}),
          ...(createSession !== undefined ? { createSession } : {}),
        });
      },
      delete: async (input: DeletePasskeyInput) => {
        await run(deletePasskey, { id: input.id });
        if (!input.credentialId) {
          return;
        }
        const rpID = resolveRpID(config);
        if (!rpID) {
          return;
        }
        try {
          const { sendSignal } = await loadWebAuthn();
          await sendSignal({
            signalName: "unknownCredential",
            rpID,
            credentialID: input.credentialId,
          });
        } catch {
          // Best-effort: browsers may not support Signal API yet.
        }
      },
      isSupported: async () => {
        const { browserSupportsWebAuthn } = await loadWebAuthn();
        return browserSupportsWebAuthn();
      },
      browserSupportsPasskeys: async () => {
        const { browserSupportsPasskeys } = await loadWebAuthn();
        return browserSupportsPasskeys();
      },
    },
  };
}
