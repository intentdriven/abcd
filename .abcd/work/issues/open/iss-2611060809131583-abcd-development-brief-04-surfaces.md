---
schema_version: 1
id: "iss-2611060809131583"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md"
remedy: "Drop the adapter clause from the review gate so it agrees with the chapter's statement that the host runs every lifeboat agent."
---

`.abcd/development/brief/04-surfaces/02-disembark.md:212-215` says an opt-in oracle adapter runs the lifeboat review when wired. No adapter can: the same chapter says at :313 that no lifeboat agent is sent to a provider, and `abcd disembark review` routed to a provider is refused at exit 2 (`internal/surface/cli/cli.go:1316`, `internal/surface/cli/dispatch.go:96`, `internal/surface/cli/dispatch.go:217`).
