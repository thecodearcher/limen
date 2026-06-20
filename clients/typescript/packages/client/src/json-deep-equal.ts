function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

function equalArrays(a: unknown[], b: unknown[]): boolean {
  if (a.length !== b.length) {
    return false;
  }

  for (let i = 0; i < a.length; i += 1) {
    if (!deepJsonEqual(a[i], b[i])) {
      return false;
    }
  }
  return true;
}

function equalObjects(a: Record<string, unknown>, b: Record<string, unknown>): boolean {
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  if (aKeys.length !== bKeys.length) {
    return false;
  }

  for (const key of aKeys) {
    if (!Object.hasOwn(b, key) || !deepJsonEqual(a[key], b[key])) {
      return false;
    }
  }
  return true;
}

/** Structural equality for JSON-shaped values. */
export function deepJsonEqual(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) {
    return true;
  }

  const aIsArray = Array.isArray(a);
  const bIsArray = Array.isArray(b);
  if (aIsArray && bIsArray) {
    return equalArrays(a, b);
  }

  if (!isPlainObject(a) || !isPlainObject(b)) {
    return false;
  }
  return equalObjects(a, b);
}
