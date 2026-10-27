---
schema_version: 1
id: "iss-2610262032521502"
slug: "launch-says-the-repo-organises-work-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 328 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:489"
remedy: "none (filed automatically)"
finding_ref: cfk-869ce71352fbb5aa
---

launch says the repo organises work by phase; the phase entry is superseded, sequencing being dependencies plus the lifecycle shelves

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:489` — "the repo organises work by **phase** (see"
- End B: `.abcd/development/brief/glossary/core/phase.md:18` — "the sequencing layer is dependencies plus the lifecycle shelves"

04-launch's versioning section says versions are an output because the repo organises work by phase, a release number falling out when a stretch of that work is published. core/phase is superseded: the sequencing layer is dependencies plus the lifecycle shelves, and the phase documents stay only as history. Launch states a retired unit as the live organiser of work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 328.
