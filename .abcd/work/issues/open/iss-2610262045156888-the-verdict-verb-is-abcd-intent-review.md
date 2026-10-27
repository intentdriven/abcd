---
schema_version: 1
id: "iss-2610262045156888"
slug: "the-verdict-verb-is-abcd-intent-review"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1262 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-80, itd-123]
finding_ref: cfk-607753d8b8340a88
---

The verdict verb is `abcd intent review ingest` in itd-80 and `abcd intent audit` in itd-123

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent review ingest` ingests a verdict."
- End B: `.abcd/development/intents/shipped/itd-123-the-intent-audit-says-audit-abcd-intent.md:21` — "`abcd intent audit` emits the same per-criterion"

itd-80's shipped scope names the verb that ingests a fidelity verdict `abcd intent review ingest`. itd-123 renames it, so that `abcd intent audit` emits the same per-criterion verdicts. One verb has two names, and anyone who types the name itd-80 gives gets a verb that has been renamed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1262.
