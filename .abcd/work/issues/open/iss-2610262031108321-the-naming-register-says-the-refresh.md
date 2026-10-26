---
schema_version: 1
id: "iss-2610262031108321"
slug: "the-naming-register-says-the-refresh"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 95 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:208"
remedy: "none (filed automatically)"
finding_ref: cfk-4aa7accdb8607ce2
---

The naming register says the refresh backstop forces a full re-inject regardless of dedup, but the configuration chapter says a domain still in force is never re-injected unchanged within a session

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:208` — "Every N prompts, the prompt-router hook forces a full re-inject regardless of dedup signature match (compaction recovery)."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1039` — "a domain that stays in force is still never re-injected unchanged within a session"

The naming register defines `rules.force_refresh_every_n` as forcing a full re-inject every N prompts regardless of the dedup signature match, as compaction recovery. The configuration chapter says compaction recovery is event-driven, on SessionStart and PreCompact, and that the counter only re-injects always-relevant domains. It also says a domain that stays in force is never re-injected unchanged within a session. A full re-inject every N prompts and no unchanged re-inject within a session cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 95.
