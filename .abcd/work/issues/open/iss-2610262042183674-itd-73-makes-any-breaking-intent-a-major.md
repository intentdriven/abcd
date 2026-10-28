---
schema_version: 1
id: "iss-2610262042183674"
slug: "itd-73-makes-any-breaking-intent-a-major"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1060 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-73-derived-versioning.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-73]
finding_ref: cfk-895b1d6ea1745549
---

itd-73 makes any breaking intent a major bump and a cut with nothing additive a patch; the version entry bumps the minor for a breaking change before 1.0 and derives no version for an empty cut.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-73-derived-versioning.md:30` — "any `breaking` → major, else any `additive` → minor, else patch"
- End B: `.abcd/development/brief/glossary/distribution/version.md:30` — "pre-1.0, a breaking change bumps the minor), and an all-internal or empty cut derives none"

itd-73's rule is any breaking intent gives a major bump, else any additive one gives a minor, else a patch, and it leaves pre-1.0 semantics out of scope. The version entry says that before 1.0 a breaking change bumps the minor, and that an all-internal or empty cut derives no version. For a 0.x release with a breaking intent, or a cut with nothing user-facing in it, the two rules give different versions.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1060.
