# Limen Roadmap

Where Limen is going, so contributors can see what's being built first-party, what's open, and what's out of scope **before writing code**. For anything non-trivial: open an issue first.

Status legend: ✅ shipped · 🚧 in progress · 📋 planned (first-party) · 🙋 help wanted · 🤔 considering — may or may not happen · ⛔ not planned

## Recently shipped

| item | notes |
|---|---|
| Access control package (`access`) | Statements, roles, permission checks. Used by organization and API keys. |
| **Organizations** (memberships, roles, invitations, active org) | Multi-tenant foundation. |
| Dynamic custom roles (runtime-defined, per-org) | Opt-in via `WithCustomRoles`. |
| API keys | Including org-scoped keys. |
| TypeScript client (`limen-auth`) | Plugin clients for credential, OAuth, 2FA, session JWT, organization, API keys. |

## Track 1: Core

| item | status | notes |
|---|---|---|
| Native relation queries (joins + relation filtering) | 🚧 | |

## Track 2: Authentication methods (plugins)

| item | status | notes |
|---|---|---|
| Passkeys / WebAuthn | 📋 | |
| Anonymous / guest sessions | 📋 | |
| Additional OAuth providers | 🙋 | Follow the `oauth-*` pattern. Safest contribution lane in the repo. |

## Track 3: Authorization & multi-tenancy

| item | status | notes |
|---|---|---|
| RBAC (global user roles) | 📋 | |
| Teams (org sub-groups) | 📋 | |
| Admin plugin (user management, impersonation, bans) | 📋 | |
| Billing / seat management | 🤔 | Undecided. |

## Track 4: Enterprise / federation

| item | status | notes |
|---|---|---|
| SSO (SAML / OIDC as a relying party, per-org) | 📋 | |
| OIDC provider (Limen as the IdP) | 📋 | |
| SCIM / directory sync | ⛔ for now | Revisit only after SSO exists. |
| Device authorization flow | 📋 | |

## Track 5: Ecosystem

| item | status | notes |
|---|---|---|
| Database adapters beyond gorm/sql | 🙋 | |
| Cache adapters beyond memory/redis | 🙋 | |

## How work gets claimed

1. 🚧 / 📋 items are first-party: contribute via issue discussion.
2. 🙋 items: comment on (or open) the tracking issue to claim before starting.
3. 🤔 items: open a discussion to help decide.
4. Anything touching the adapter contract, schema system, or plugin interface should start as a design discussion in an issue before code.
