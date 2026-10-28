---
schema_version: 1
id: "iss-2610262039150797"
slug: "itd-57-uses-voyage-for-a-project-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 848 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-57]
finding_ref: cfk-bbe7d73a3c7ac9ef
---

itd-57 uses 'voyage' for a project's configured set of specs, but the configuration page defines a voyage as the user-scope log of disembark runs

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:13` — "Every abcd Voyage Ships With A Manual-Hold Sentinel"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:494` — "`voyage/` is **operations** — the verb, what we did — as distinct from the"

itd-57's title and press release treat the voyage as the project's own sequence of specs, whose first spec is planted at configure time. The configuration page defines `voyage/` as operations: the user-scope, never-committed log of disembark runs, distinct from the lifeboat. The same word names two different things.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 848.
