/** Collapse an intersection into a single readable object type on hover. */
export type Prettify<T> = { [K in keyof T]: T[K] } & {};

/** Turn a union `A | B | C` into the intersection `A & B & C`. */
export type UnionToIntersection<U> = (U extends unknown ? (x: U) => void : never) extends (x: infer I) => void
  ? I
  : never;

/**
 * Convert a kebab-case literal to camelCase: `"magic-link"` -> `"magicLink"`.
 */
export type KebabToCamel<S extends string> = S extends `${infer Head}-${infer Tail}`
  ? `${Head}${Capitalize<KebabToCamel<Tail>>}`
  : S;

/** Split a string literal into a tuple on a delimiter, dropping empty segments. */
export type Split<S extends string, D extends string> = S extends `${infer Head}${D}${infer Tail}`
  ? Head extends ""
    ? Split<Tail, D>
    : [Head, ...Split<Tail, D>]
  : S extends ""
    ? []
    : [S];

/** Check if a type is `any`. */
export type IsAny<T> = 0 extends 1 & T ? true : false;
