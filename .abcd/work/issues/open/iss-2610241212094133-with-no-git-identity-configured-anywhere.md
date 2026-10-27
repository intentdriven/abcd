---
schema_version: 1
id: "iss-2610241212094133"
slug: "with-no-git-identity-configured-anywhere"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "setup redesign build, identity lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "the pre-commit identity gate (branch feat/setup-redesign-identity)"
remedy: "detect git's invented identity in the hook (git var GIT_AUTHOR_IDENT against user.email being unset in every scope) and refuse it as unset, matching the Go check"
---

with no git identity configured anywhere, git invents one from the account and host name and exports it into the pre-commit hook, where it reads exactly like an identity a launcher supplied, so the per-machine identity gate lets an invented identity commit when no record exists; only the Go check reports it, as git_identity.unset.
