---
schema_version: 1
id: "iss-2610262032208043"
slug: "capture-says-the-release-guard-never"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 254 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:334"
remedy: "Reword 06-capture.md lines 332-335: the deferral verb refuses a record graded below major, but an absent, misspelled or out-of-enum grade does block the cut (gradeBlocks in internal/core/changelog/findings.go), so such a record is graded first and then deferred; optionally say the same in 04-launch.md lines 585-588."
finding_ref: cfk-7111704d5c136f3a
---

Capture says the release guard never blocks a record graded below major; launch says it also blocks an absent or unreadable grade

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:334` — "a record whose grade is neither `major` nor `critical`, which the guard never blocks on"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:587` — "is a severity of `major` or `critical`, and also a severity that is absent, misspelled, or outside the ledger's enum"

Launch's unfixed-findings guardrail blocks on a major or critical grade and also on a severity that is absent, misspelled or outside the enum, and names deferral with the waiver pair as one of its routes out. Capture's deferral verb refuses every record whose grade is neither major nor critical, on the stated premise that the guard never blocks on such a record. Both cannot hold: an ungraded record does block the cut, yet the verb that writes the waiver refuses it, so the deferral route launch offers is closed for exactly the records launch counts as unjudged.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 254.
