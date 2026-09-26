---
schema_version: 1
id: "iss-2609261004260522"
slug: "lab-sweep-is-not-fail-closed-ac3-state-probes-is-skipped"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lab"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lab/sweep.go"
resolution: "The sweep reads every probe file but the five capture files, lists every document it could not read (over the cap, NUL in the head, not a regular file) by path, and fails as sweep/unswept while any correction is recorded (TestSweepReadsAProbeRecordButNotItsCaptureFiles, TestSweepFailsOnADocumentItCannotRead)."
impact: fix
resolved_by:
  commit: "121f6a31"
---

lab sweep is not fail-closed (AC3): state/probes/ is skipped whole, so a retracted claim in a probe's record.md Observation is never listed; a document over the 4 MiB read cap is listed as not swept yet the sweep still passes; and a document with a NUL in its first 8000 bytes is skipped silently, not even listed. Sweep every probe file but the five capture files, list every document not swept by path, and fail the sweep when corrections exist and any document was not swept.

## Grounds

- pursued: a retracted literal in a probe's record.md is listed, and an over-cap, NUL-headed or linked document halts the sweep naming it; a lab document the walk never reaches and never lists would show it wrong
