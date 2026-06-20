# limen-ts

TypeScript client SDK for [Limen](https://github.com/thecodearcher/limen) — a modern, composable authentication library for Go.

## Packages

- [`limen-auth`](./packages/client) — framework-agnostic core
- [`limen-auth/react`](./packages/client/src/react) — React hooks adapter (subpath of `limen-auth`)
- [`limen-auth/vue`](./packages/client/src/vue) — Vue composables adapter (subpath of `limen-auth`)
- [`limen-auth/svelte`](./packages/client/src/svelte) — Svelte stores adapter (subpath of `limen-auth`)
- [`limen-auth/solid`](./packages/client/src/solid) — Solid primitives adapter (subpath of `limen-auth`)

## Development

```bash
pnpm install
pnpm typecheck
pnpm lint
pnpm test
pnpm build
```
