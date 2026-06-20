import { describe, expect, it } from "vitest";
import { deepJsonEqual } from "../src/json-deep-equal";

describe("deepJsonEqual", () => {
  it("compares arrays by value, order-sensitive", () => {
    expect(deepJsonEqual([1, 2, 3], [1, 2, 3])).toBe(true);
    expect(deepJsonEqual([{ a: 1 }], [{ a: 1 }])).toBe(true);
    expect(deepJsonEqual([1, 2], [2, 1])).toBe(false);
    expect(deepJsonEqual([1, 2], [1, 2, 3])).toBe(false);
  });

  it("compares objects independent of key order", () => {
    expect(deepJsonEqual({ a: 1, b: 2 }, { b: 2, a: 1 })).toBe(true);
    expect(deepJsonEqual({ a: 1 }, { a: 1, b: 2 })).toBe(false);
    expect(deepJsonEqual({ a: 1 }, { a: 2 })).toBe(false);
    expect(deepJsonEqual({ a: 1 }, { b: 1 })).toBe(false);
  });

  it("compares nested objects with array fields", () => {
    const session = { user: { id: "u1", roles: ["admin", "member"], meta: { plan: "pro" } } };
    expect(deepJsonEqual(session, structuredClone(session))).toBe(true);
    expect(deepJsonEqual(session, { user: { id: "u1", roles: ["member", "admin"], meta: { plan: "pro" } } })).toBe(
      false,
    );
  });

  it("distinguishes arrays from objects and rejects non-JSON structures", () => {
    expect(deepJsonEqual([], {})).toBe(false);
    expect(deepJsonEqual({ 0: "a" }, ["a"])).toBe(false);
    expect(deepJsonEqual(new Date("2026-06-20T00:00:00Z"), new Date("2026-06-20T00:00:00Z"))).toBe(false);
    expect(deepJsonEqual(() => "x", () => "x")).toBe(false);
  });
});
