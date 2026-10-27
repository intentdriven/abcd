---
schema_version: 1
id: "iss-2610262030345408"
slug: "the-launch-chapter-says-the-repo"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 17 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:489"
remedy: "In 04-launch.md section 3 (lines 487-491) replace 'the repo organises work by phase (adr-9)' with sequencing by dependencies plus the lifecycle shelves (adr-2609292012006845, carrying forward adr-2609212115255771), keeping the point that a version is an output of the cut; retarget the Phase ownership note at line 64 in the same edit."
finding_ref: cfk-8c0e2d349347a169
---

The launch chapter says the repo organises work by phase per adr-9; the mental model says the phase is retired and adr-9 superseded

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:489` — "the repo organises work by **phase** (see [adr-9] (../../decisions/adrs/0009-phase-as-product-layer.md)), and a release number is what falls out when a stretch of that work is published."
- End B: `.abcd/development/brief/01-product/03-mental-model.md:56` — "The phase and the milestone are retired"

The launch chapter's versioning section says the repo organises work by phase, citing adr-9, and that a release number falls out when a stretch of that work is published. The mental model says the phase and the milestone are retired, and its opening paragraph names adr-9 as superseded. The launch chapter's account of what a version measures rests on a unit the record no longer has.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 17.
