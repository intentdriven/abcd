---
schema_version: 1
id: "iss-2610262032060007"
slug: "the-register-gives-the-forced-rules"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 221 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:208"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-14487ee981bacdee
---

The register gives the forced rules refresh a default of 15 prompts; itd-3 says it defaults to 5.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:208` — "Integer; default 15 (`rules.DefaultRefreshBackstop`)"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "force full re-inject every N turns (N defaults to 5, configurable)"

The reserved-vocabulary row for `force_refresh_every_n` states a default of 15 (`rules.DefaultRefreshBackstop`). itd-3 states that the full re-inject runs every N turns with N defaulting to 5. The two give different defaults for one setting.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 221.
