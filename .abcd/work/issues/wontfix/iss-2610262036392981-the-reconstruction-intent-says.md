---
schema_version: 1
id: "iss-2610262036392981"
slug: "the-reconstruction-intent-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 664 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609091718595846-any-captured-session-can-be-handed-to-an.md:57"
remedy: "none (filed automatically)"
related_intents: [itd-2609091718595846]
finding_ref: cfk-d90b00ba4a18522a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The chapter itself separates adding to the corpus from the seam's directory creation and legacy move."
---

The reconstruction intent says reconstruction makes no write to the store; the history chapter says every history verb's resolve seam creates the store chain and moves a legacy corpus.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2609091718595846-any-captured-session-can-be-handed-to-an.md:57` — "Reconstruction reads records and writes only its two output files."
- End B: `.abcd/development/brief/04-surfaces/11-history.md:186` — "Every history verb reaches the store through one resolve seam, and that seam creates the store chain when it is absent and moves"

The intent puts any write to the store out of scope and says reconstruction reads records and writes only its artefact and its telemetry file. The history chapter says every history verb reaches the store through one resolve seam. That seam creates the store chain when it is absent and moves a corpus left at the legacy location into it, leaving a tombstone. `history reconstruct` is one of those verbs, so either it writes to the store or the seam statement does not hold for it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 664.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The chapter itself separates adding to the corpus from the seam's directory creation and legacy move.
