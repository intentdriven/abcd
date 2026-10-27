---
schema_version: 1
id: "iss-2610262031549181"
slug: "itd-147-s-press-release-says-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 193 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-147]
finding_ref: cfk-35e556161f829a57
---

itd-147's press release says the generated appendix carries exit codes, schema fields and counts; invariant 18 says flags and sub-verbs only.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:24` — "sub-verbs, exit codes, schema fields and counts, derived from the command tree"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:173` — "The appendix carries flags and sub-verbs only: exit codes, output fields and behavioural claims stay prose and review-grain"

itd-147's press release says each surface chapter's generated block carries the verb's flags, 'sub-verbs, exit codes, schema fields and counts, derived from the command tree'. Invariant 18 states 'The appendix carries flags and sub-verbs only: exit codes, output fields and behavioural claims stay prose and review-grain'. The intent's own decisions narrow to flags and sub-verbs first, but its press release, the part a reader takes as the promise, still claims a wider shipped block than the invariant allows.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 193.
