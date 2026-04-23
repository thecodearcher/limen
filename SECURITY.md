# Security Policy

We take the security of Limen and its users seriously. If you believe you've
found a security vulnerability in Limen, please report it to us privately so we
can investigate and address it before it is publicly disclosed.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues,
discussions, or pull requests.**

Instead, report them by email to:

**[security@limenauth.dev](mailto:security@limenauth.dev)**

To help us triage and respond quickly, please include as much of the following
as you can:

- A description of the issue and its potential impact
- Steps to reproduce, or a proof-of-concept
- The affected version(s), module(s), or plugin(s) (e.g. `limen`,
  `adapters/gorm`, `plugins/credential-password`)
- Any relevant logs, stack traces, or configuration
- Your name / handle if you'd like to be credited in the advisory

Please give us a reasonable window to address the issue before any public
disclosure.

## Scope

In scope:

- The `limen` core library
- First-party adapters under `adapters/`
- First-party plugins under `plugins/`
- Official examples under `examples/` where they demonstrate insecure defaults

Out of scope:

- Vulnerabilities in third-party dependencies (please report those upstream;
  if Limen's usage of a dependency makes the issue exploitable, that _is_ in
  scope)
- Issues that require a compromised host, stolen secret, or attacker-controlled
  build environment
- Missing security hardening that does not correspond to a concrete,
  reproducible vulnerability

Thank you for helping keep Limen and its users safe.
