---
schema_version: 1
id: "iss-2610262037477814"
slug: "itd-147-rules-that-a-surface-chapter-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 749 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:90"
remedy: "none (filed automatically)"
related_intents: [itd-147]
finding_ref: cfk-c1001f213aa97669
---

itd-147 rules that a surface chapter's hand-written prose never states a sub-verb; 27-implement states its sub-verbs above the generated appendix in a table and in prose

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:90` — "the prose above it never states a flag or sub-verb"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:498` — "The outage sub-verb renders it, and its own sub-verbs report a loss"

Shipped itd-147 makes the generated appendix the only place a chapter states flags and sub-verbs: the prose above it never states a flag or sub-verb. 27-implement keeps a hand-maintained sub-verb table above the appendix marker. The table lists join through outage clear, with outage record, probe, ack and clear among them. Its prose also names sub-verbs directly, for example the outage sub-verb and its own sub-verbs. Each sub-verb is therefore stated twice, once by hand and once generated. That is the duplication itd-147 says it removes by construction.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 749.
