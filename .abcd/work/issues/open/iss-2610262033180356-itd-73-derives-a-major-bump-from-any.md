---
schema_version: 1
id: "iss-2610262033180356"
slug: "itd-73-derives-a-major-bump-from-any"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 388 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-73-derived-versioning.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-73]
finding_ref: cfk-d20de0eb5de551e8
---

itd-73 derives a major bump from any breaking intent; 04-launch says no set of records derives 1.0.0 from a 0.y.z base

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-73-derived-versioning.md:30` — "take the highest-severity `impact`, and compute the next SemVer — any `breaking` → major, else any `additive` → minor, else patch"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:534` — "No set of records derives `1.0.0` from a `0.y.z` base."

itd-73 states its derivation rule with no condition: any `breaking` impact is a major bump. 04-launch's bump-tier table maps `breaking` to a minor bump before 1.0.0 and says no set of records derives `1.0.0` from a `0.y.z` base. At a 0.y.z version, a breaking intent gives 1.0.0 under itd-73 and a minor bump under 04-launch, so the two rules cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 388.
