import { expectTypeOf, test } from "vitest";
import { createAuthClient } from "../src";
import { sessionJwtPlugin } from "../src/plugins/session-jwt";

test("session-jwt actions stay on the client when the only route is expose:false", () => {
  const auth = createAuthClient({
    baseURL: "http://localhost:8080",
    plugins: [sessionJwtPlugin()],
  });

  expectTypeOf(auth.sessionJwt.getAccessToken).toBeFunction();
  expectTypeOf(auth.sessionJwt.refresh).toBeFunction();
  expectTypeOf(auth.sessionJwt.getTokens).toBeFunction();
  expectTypeOf(auth.sessionJwt.clear).toBeFunction();
});
