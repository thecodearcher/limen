import { expectTypeOf, test } from "vitest";
import type { Session } from "../src";
import { createAuthClient } from "../src";
import type { Passkey } from "../src/plugins/passkey";
import { passkey } from "../src/plugins/passkey";

test("passkey client surface", () => {
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [passkey({ rpID: "localhost" })],
  });

  expectTypeOf(auth.signIn.passkey).toBeFunction();
  expectTypeOf(auth.passkey.register).toBeFunction();
  expectTypeOf(auth.passkey.list).toBeFunction();
  expectTypeOf(auth.passkey.update).toBeFunction();
  expectTypeOf(auth.passkey.delete).toBeFunction();
  expectTypeOf(auth.passkey.beginRegistration).toBeFunction();
  expectTypeOf(auth.passkey.finishRegistration).toBeFunction();
  expectTypeOf(auth.passkey.beginAuthentication).toBeFunction();
  expectTypeOf(auth.passkey.finishAuthentication).toBeFunction();
  expectTypeOf(auth.passkey.isSupported).returns.resolves.toBeBoolean();
  expectTypeOf(auth.passkey.browserSupportsPasskeys).returns.resolves.toBeBoolean();
  expectTypeOf(auth.passkey.register).returns.resolves.toEqualTypeOf<Passkey | Session>();
});
