---
schema_version: 1
id: "iss-2610262041210479"
slug: "the-transcript-capture-draft-takes-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 994 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:131"
remedy: "none (filed automatically)"
related_intents: [itd-59]
finding_ref: cfk-630f65fdba9ae370
---

The transcript-capture draft takes as given an autonomous worker running on the pluggable run seam; the build sequence keeps the run seam a design target.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:131` — "Autonomous run seam *(design target)*"
- End B: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19` — "the autonomous worker on the pluggable run seam"

itd-59 starts from an autonomous worker on the pluggable run seam that runs for hours, with plan/work/completion-review phases and per-pass logs that get pruned. 01-build-sequence.md marks the autonomous run seam a design target that does not ship. The worker whose transcripts the draft captures is not on the brief's map.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 994.
