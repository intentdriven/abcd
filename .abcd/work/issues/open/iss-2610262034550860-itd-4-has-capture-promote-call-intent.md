---
schema_version: 1
id: "iss-2610262034550860"
slug: "itd-4-has-capture-promote-call-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 533 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-4-issue-capture.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-614971807885a7c4
---

itd-4 has capture promote call intent new with the issue's body; the capture surface page says promotion mints the draft natively with a by-id pointer, not a copy.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-4-issue-capture.md:35` — "promote an `iss-N` entry to an intent draft (calls `/abcd:intent new` with the entry body + bidirectional link)"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:186` — "with the slug reused and the body a by-id pointer rather than a copy"

itd-4 scopes `capture promote` as calling `/abcd:intent new` with the entry body and a bidirectional link. 06-capture.md says one invocation mints the draft itself, with the slug reused and the body a by-id pointer rather than a copy, and that this native promotion superseded an earlier command-orchestrated flow; `intent new` was also removed by itd-2609212130136102. A promoted draft cannot both carry the issue's body through intent new and only point at the issue by id.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 533.
