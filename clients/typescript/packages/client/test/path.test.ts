import { describe, expect, it } from "vitest";
import { chainFromDotted, pathToChain, resolvePath } from "../src/path";

describe("pathToChain", () => {
  it("camelCases kebab segments and drops empties", () => {
    expect(pathToChain("/me")).toEqual(["me"]);
    expect(pathToChain("/signin/credential")).toEqual(["signin", "credential"]);
    expect(pathToChain("/revoke-sessions")).toEqual(["revokeSessions"]);
    expect(pathToChain("/otp/send")).toEqual(["otp", "send"]);
  });

  it("drops `:param` segments (params come from input, not the chain)", () => {
    expect(pathToChain("/:provider/authorize")).toEqual(["authorize"]);
    expect(pathToChain("/:provider/tokens/refresh")).toEqual(["tokens", "refresh"]);
  });
});

describe("chainFromDotted", () => {
  it("splits absolute `as` chains", () => {
    expect(chainFromDotted("twoFactor.getTotpUri")).toEqual(["twoFactor", "getTotpUri"]);
    expect(chainFromDotted("passwords.set")).toEqual(["passwords", "set"]);
  });
});

describe("resolvePath", () => {
  it("passes through when there are no params", () => {
    const input = { a: 1 };
    expect(resolvePath("/accounts", undefined, input)).toEqual({ path: "/accounts", rest: input });
    expect(resolvePath("/accounts", [], input)).toEqual({ path: "/accounts", rest: input });
  });

  it("substitutes params and strips them from the payload", () => {
    const { path, rest } = resolvePath("/:provider/authorize", ["provider"], {
      provider: "google",
      redirectUri: "https://app/cb",
    });
    expect(path).toBe("/google/authorize");
    expect(rest).toEqual({ redirectUri: "https://app/cb" });
  });

  it("encodes param values", () => {
    const { path } = resolvePath("/:provider/tokens", ["provider"], { provider: "we/ird id" });
    expect(path).toBe("/we%2Fird%20id/tokens");
  });

  it("throws when a declared param is missing", () => {
    expect(() => resolvePath("/:provider/unlink", ["provider"], {})).toThrow(/provider/);
  });
});
