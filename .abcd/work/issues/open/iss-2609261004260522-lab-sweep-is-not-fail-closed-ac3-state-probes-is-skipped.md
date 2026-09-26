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
---

lab sweep is not fail-closed (AC3): state/probes/ is skipped whole, so a retracted claim in a probe's record.md Observation is never listed; a document over the 4 MiB read cap is listed as not swept yet the sweep still passes; and a document with a NUL in its first 8000 bytes is skipped silently, not even listed. Sweep every probe file but the five capture files, list every document not swept by path, and fail the sweep when corrections exist and any document was not swept.
