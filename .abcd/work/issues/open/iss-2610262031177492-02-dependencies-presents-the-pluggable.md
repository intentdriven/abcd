---
schema_version: 1
id: "iss-2610262031177492"
slug: "02-dependencies-presents-the-pluggable"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 111 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:45"
remedy: "none (filed automatically)"
finding_ref: cfk-10ae02a1b7f82729
---

02-dependencies presents the pluggable run seam, with a native loop as its fallback, as a present native default; the build sequence marks the run seam a design target that does not ship.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:45` — "the native loop is the always-available fallback"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:131` — "Autonomous run seam *(design target)*"

02-dependencies says every capability once delegated to a bundled tool now has a native default and lists the autonomous run as a pluggable run seam in which 'the native loop is the always-available fallback'. The build sequence heads the same seam 'Autonomous run seam *(design target)*' and says the seam does not ship: no run verb is registered and the run adapter is on the planned-seams list. 02-dependencies states as current what the build sequence records as unbuilt, without the staged mark the truth rule requires.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 111.
