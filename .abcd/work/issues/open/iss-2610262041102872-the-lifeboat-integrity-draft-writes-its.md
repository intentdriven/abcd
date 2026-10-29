---
schema_version: 1
id: "iss-2610262041102872"
slug: "the-lifeboat-integrity-draft-writes-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 982 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:178"
remedy: "Re-home the verification report under the gitignored local tier (run output, per the 2026-07-12 adjudication) instead of the retired logbook path, and check its /abcd:audit surface name against itd-124, which named the conformance check lint."
related_intents: [itd-35]
finding_ref: cfk-a3780ea149973f90
---

The lifeboat-integrity draft writes its report under `.abcd/logbook/`, a location the build sequence has retired and a detector fails the build over.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:178` — "`TestNoRetiredLogbookLocationInSource` fails the build if any non-test Go source"
- End B: `.abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:42` — "`.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}`"

01-build-sequence.md says `.abcd/logbook/` is retired and that `TestNoRetiredLogbookLocationInSource` fails the build if non-test Go source under internal/ names it. itd-35 puts its verification report at `.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}`, so building it as written would trip the detector.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 982.
