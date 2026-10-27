---
schema_version: 1
id: "iss-2610262044160347"
slug: "the-dependency-graph-says-severity"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1195 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-78, itd-2609211116005482]
finding_ref: cfk-a289fd331837c447
---

The dependency graph says severity inheritance decides what to build first, but build next picks readiest first and explicitly not most-important-first

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17` — "A minor intent holding up a major one jumps the queue while staying honestly minor"
- End B: `.abcd/development/intents/shipped/itd-2609211116005482-abcd-build-next-picks-the-readiest.md:66` — "Not oldest-first, not most-important-first; the evidence for readiness as the predictor is the reason"

itd-78 claims its derived effective priority computes the order, so a minor blocker of a major intent jumps the queue. itd-2609211116005482 rules that the pick is readiest first, oldest among equals, and not most-important-first, and it filters on edges without ranking on them. The two records disagree on what decides the next thing built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1195.
