---
schema_version: 1
id: "iss-2610262039439681"
slug: "03-configuration-makes-the-rules-refresh"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 881 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:93"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-e332ef7c41019880
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/rules/inject.go has DefaultRefreshBackstop = 15 with SessionStart/PreCompact reset as primary; the shipped intent text predates the D1 ruling made during the same build."
---

03-configuration makes the rules refresh event-driven with a 15-turn backstop; itd-3 forces a full re-inject every N turns, N defaulting to 5.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:93` — "prompt-router refresh backstop (itd-3). The primary refresh is"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "Signature-based dedup; force full re-inject every N turns (N defaults to 5, configurable)."

03-configuration's `rules.force_refresh_every_n` defaults to 15. It calls the primary refresh event-driven, on SessionStart and PreCompact, and says a domain that stays in force is never re-injected unchanged within a session. itd-3 specifies signature-based dedup with a forced full re-inject every N turns, N defaulting to 5, as the way to recover from compaction. The default and the refresh mechanism disagree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 881.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/rules/inject.go has DefaultRefreshBackstop = 15 with SessionStart/PreCompact reset as primary; the shipped intent text predates the D1 ruling made during the same build.
