---
schema_version: 1
id: "iss-2610262035168556"
slug: "itd-73-derives-the-release-version-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 560 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-73-derived-versioning.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-73]
finding_ref: cfk-35611e742150dfc6
---

itd-73 derives the release version from the shipped intents alone; the capture chapter says a resolved issue's impact drives the derived version too

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-73-derived-versioning.md:30` — "gather the intents shipped since the previous release, take the highest-severity `impact`, and compute the next SemVer"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:405` — "required in resolved/; drives the derived version and changelog inclusion"

itd-73 computes the next SemVer by gathering the intents shipped since the previous release and taking their highest impact. The capture chapter says an issue's impact, required in resolved/, drives the derived version and changelog inclusion. Take a release with a breaking resolved issue and no breaking intent: itd-73 would not bump the major version, and the capture chapter would. The two documents name different inputs for one computation.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 560.
