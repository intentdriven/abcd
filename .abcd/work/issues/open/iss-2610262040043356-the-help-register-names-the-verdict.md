---
schema_version: 1
id: "iss-2610262040043356"
slug: "the-help-register-names-the-verdict"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 907 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:152"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-f9ce9d52438c2a5c
---

The help register names the verdict ingest `intent audit ingest`; itd-80 names it `intent review ingest`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:152` — "`intent audit ingest`, `intent consistency ingest`, `intent prepass`"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent review ingest` ingests a verdict."

The README's help table lists `intent audit ingest` beside `intent consistency ingest` as the verbs agents and hosts call. itd-80, which delivered the fidelity-verdict ingest, names it `abcd intent review ingest`. That is two names for one verb, and someone building from itd-80 would type a verb the help does not list.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 907.
