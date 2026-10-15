---
id: dcn-2607100000000006
slug: repo-preparation-is-a-plugin-skill-abcd
date: 2026-07-10
---
2026-07-10: Repo preparation is a plugin skill (`/abcd:prepare-this-repo`),
superseding the external scaffold-repo script's entry point. Grilled rulings:
the committed AGENTS.md working-conventions section is full-inline and
NAMELESS (a pre-public repo name never lands in target repos) between dated
markers for later tooling; the skill hard-refuses not-owned repos (no audit,
no local layer — we don't impose our principles on others' repos); legacy
root `.work/` layouts migrate propose-then-sign-off, never leaving two
working-state homes; no re-run/update machinery now — the CLI will own
managed-repo migration (gaps seeded as iss-84/iss-85, originally minted as
duplicate iss-56/iss-57). Rejected: a standalone
handover prompt file (drifts, unversioned); naming abcd in private-only
target repos (two-class rule someone eventually gets wrong).
