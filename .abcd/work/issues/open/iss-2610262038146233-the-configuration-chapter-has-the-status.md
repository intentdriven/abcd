---
schema_version: 1
id: "iss-2610262038146233"
slug: "the-configuration-chapter-has-the-status"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 781 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:1021"
remedy: "Edit 03-configuration.md (The status block paragraph): replace the 'each item once, newest first' summary with the fixed order and caps, or cite 39-status.md alone and drop the restatement."
finding_ref: cfk-552e727474e5d118
---

The configuration chapter has the STATUS block show each item once, newest first; the status chapter shows them in a fixed order, sums up the doubts and keeps only the newest announcement per item.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:1021` — "doubtful branch and failed fetch, once, newest first, at most ten lines and a"
- End B: `.abcd/development/brief/04-surfaces/39-status.md:120` — "at most two one by one and otherwise one line that sums them up and always"

03-configuration says the prompt router's `## STATUS` block shows each new announcement, release hold, doubtful branch and failed fetch once, newest first. 39-status sets a fixed order (failed fetches, then release holds, then the newest announcement per item, then the doubts), shows at most two doubts one by one and sums up the rest in one line. Showing every doubtful branch newest first and summing up the doubts after two are different outputs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 781.
