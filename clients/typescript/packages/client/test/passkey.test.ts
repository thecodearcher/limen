import { beforeEach, describe, expect, it, vi } from "vitest";
import { createAuthClient } from "../src";
import { passkey } from "../src/plugins/passkey";
import { mockFetch, type MockReply, type Recorded } from "./helpers";

const userBody = {
  user: { id: "u1", email: "ada@example.com", email_verified_at: null, first_name: "Ada" },
};

const creationOptions = {
  challenge: "challenge",
  rp: { name: "Limen", id: "localhost" },
  user: { id: "user-id", name: "ada@example.com", displayName: "Ada" },
  pubKeyCredParams: [{ type: "public-key", alg: -7 }],
};

const requestOptions = {
  challenge: "challenge",
  rpId: "localhost",
  allowCredentials: [] as { type: string; id: string }[],
  userVerification: "preferred",
};

const attestation = {
  id: "cred-id",
  rawId: "cred-id",
  type: "public-key" as const,
  clientExtensionResults: {},
  response: {
    clientDataJSON: "cd",
    attestationObject: "ao",
    transports: ["internal"],
  },
};

const assertion = {
  id: "cred-id",
  rawId: "cred-id",
  type: "public-key" as const,
  clientExtensionResults: {},
  response: {
    clientDataJSON: "cd",
    authenticatorData: "ad",
    signature: "sig",
    userHandle: "uh",
  },
};

const passkeyBody = {
  id: "pk-1",
  name: "MacBook",
  credential_id: "cred-abc",
  transports: ["internal"],
  aaguid: null,
  backup_eligible: true,
  backup_state: false,
  last_used_at: null,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

const startRegistration = vi.fn(async () => attestation);
const startAuthentication = vi.fn(async () => assertion);
const sendSignal = vi.fn(async () => undefined);
const browserSupportsWebAuthn = vi.fn(() => true);
const browserSupportsPasskeys = vi.fn(() => true);

vi.mock("@simplewebauthn/browser", () => ({
  startRegistration,
  startAuthentication,
  sendSignal,
  browserSupportsWebAuthn,
  browserSupportsPasskeys,
}));

function setup(reply: (req: Recorded) => MockReply) {
  const { impl, calls } = mockFetch(reply);
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [passkey({ rpID: "localhost" })],
    fetchOptions: { impl },
    crossTabSync: false,
    refetchOnWindowFocus: false,
  });
  return { auth, calls };
}

describe("passkey plugin", () => {
  beforeEach(() => {
    startRegistration.mockClear().mockResolvedValue(attestation);
    startAuthentication.mockClear().mockResolvedValue(assertion);
    sendSignal.mockClear().mockResolvedValue(undefined);
    browserSupportsWebAuthn.mockClear().mockReturnValue(true);
    browserSupportsPasskeys.mockClear().mockReturnValue(true);
  });

  it("signIn.passkey runs the ceremony and stores the session", async () => {
    const { auth, calls } = setup((req) => {
      if (req.url.includes("/begin-authentication")) {
        return { body: { publicKey: requestOptions } };
      }
      return { body: userBody };
    });

    const session = await auth.signIn.passkey({ extensions: ["prf", "credProps"] });

    expect(calls[0]?.url).toBe(
      "http://localhost:8080/auth/passkeys/begin-authentication?extensions=prf%2CcredProps",
    );
    expect(startAuthentication).toHaveBeenCalledWith({
      optionsJSON: requestOptions,
      useBrowserAutofill: undefined,
      verifyBrowserAutofillInput: undefined,
    });
    expect(calls[1]?.url).toBe("http://localhost:8080/auth/passkeys/finish-authentication");
    expect(calls[1]?.body).toEqual(assertion);
    expect(session.user.id).toBe("u1");
    expect(auth.$session.get().data?.user.id).toBe("u1");
  });

  it("passkey.register with createSession returns a session", async () => {
    const { auth, calls } = setup((req) => {
      if (req.url.includes("/begin-registration")) {
        return { body: { publicKey: creationOptions } };
      }
      return { body: userBody };
    });

    const result = await auth.passkey.register({
      name: "Laptop",
      authenticatorAttachment: "platform",
      context: "ada@example.com",
      createSession: true,
      extensions: ["credProps"],
    });

    expect(calls[0]?.url).toContain("authenticator_attachment=platform");
    expect(calls[0]?.url).toContain("extensions=credProps");
    expect(calls[1]?.body).toMatchObject({
      rawId: "cred-id",
      name: "Laptop",
      create_session: true,
    });
    expect(result).toHaveProperty("user");
    expect(auth.$session.get().data?.user.id).toBe("u1");
  });

  it("passkey.register without createSession returns a camelized Passkey", async () => {
    const { auth } = setup((req) => {
      if (req.url.includes("/begin-registration")) {
        return { body: { publicKey: creationOptions } };
      }
      return { body: passkeyBody };
    });

    const result = await auth.passkey.register({ name: "Key" });

    expect(result).toMatchObject({
      id: "pk-1",
      credentialId: "cred-abc",
      backupEligible: true,
    });
    expect(auth.$session.get().data).toBeNull();
  });

  it("passkey.delete signals unknownCredential after a successful DELETE", async () => {
    const { auth, calls } = setup(() => ({ status: 204 }));

    await auth.passkey.delete({ id: "pk-1", credentialId: "cred-abc" });

    expect(calls[0]).toMatchObject({
      method: "DELETE",
      url: "http://localhost:8080/auth/passkeys/pk-1",
    });
    expect(sendSignal).toHaveBeenCalledWith({
      signalName: "unknownCredential",
      rpID: "localhost",
      credentialID: "cred-abc",
    });
  });

  it("passkey.delete is best-effort for sendSignal and skips it without credentialId", async () => {
    const { auth } = setup(() => ({ status: 204 }));

    await auth.passkey.delete({ id: "pk-1" });
    expect(sendSignal).not.toHaveBeenCalled();

    sendSignal.mockRejectedValueOnce(new Error("signal unsupported"));
    await expect(auth.passkey.delete({ id: "pk-1", credentialId: "cred-abc" })).resolves.toBeUndefined();
  });
});
