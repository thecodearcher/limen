<p align="center">
  <a href="https://limenauth.dev">
    <img src="https://raw.githubusercontent.com/thecodearcher/limen/master/banner.svg" alt="Limen" width="640" />
  </a>
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/limen-auth"><img src="https://img.shields.io/npm/v/limen-auth?style=flat&colorA=000000&colorB=000000&logo=npm&logoColor=white" alt="npm version" /></a>
  <a href="https://github.com/thecodearcher/limen"><img src="https://img.shields.io/github/stars/thecodearcher/limen?style=flat&colorA=000000&colorB=000000&logo=github" alt="GitHub stars" /></a>
</p>

## Limen Auth

Official TypeScript client SDK for **[Limen](https://github.com/thecodearcher/limen)** — a modern, composable authentication library for Go. Framework-agnostic core with first-class **React, Vue, Svelte, and Solid** adapters.

## Install

```bash
npm install limen-auth
```

## Quick start

```ts
import { createAuthClient } from "limen-auth";
import { credentialPasswordPlugin } from "limen-auth/plugins/credential";

export const auth = createAuthClient({
  baseURL: "http://localhost:8080", // your Limen server origin
  plugins: [credentialPasswordPlugin()],
});

// `auth.$session` is a reactive store for the current user — it loads on its
// own, stays in sync across tabs, and updates whenever you sign in or out.
auth.$session.subscribe(({ data, isPending }) => {
  if (isPending) return;
  console.log(data ? `Signed in as ${data.user.email}` : "Signed out!");
});

// Mutations update `$session` automatically — no manual refetch.
await auth.signIn.credential({ credential: "ada@example.com", password: "secret" });
await auth.signout();
```

Using a framework? `limen-auth/react`, `/vue`, `/svelte`, and `/solid` give you a `useSession()` hook over the same store.

## Documentation

Full guides, framework adapters (React, Vue, Svelte, Solid), the plugin catalog, and the complete API reference live at **[limenauth.dev](https://limenauth.dev)**.
