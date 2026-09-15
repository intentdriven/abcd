---
schema_version: 1
id: "iss-2608311145258479"
slug: "the-definition-locator-checks-that-a-regime-is-a-known-value"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "itd-184 fidelity audit rcp-426034d44293"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/definitions.go"
resolution: "The locator now refuses a definition whose regime disagrees with the position its filename holds, rather than resolving it. Membership in the four and agreement with the position are two questions, and the locator is the one thing that claims to resolve a position to its regime, so a confident wrong answer there hands the ingest verb the wrong licence with nothing refusing."
impact: fix
resolved_by:
  intent: "itd-185"
  commit: "3ed2e773"
---

The definition locator checks that a regime is a KNOWN value but never that it AGREES with the position it sits under. knownRegime tests membership in the four, so a definition carrying regime evaluative under position widening loads clean: the binary prints all four definitions and exits 0 while the byte-identity and pinning tests go red. The comment at the top of definitions.go argues that answering from a table rather than the file is what would make a drift undetectable, but the file is not checked against the position either, so the drift is undetectable at RUNTIME and caught only in CI. This matters downstream rather than here: spc-63's regime gate resolves a run's position to its definition and reads the regime from LoadDefinition as the source of truth, so a drifted file hands the ingest verb the wrong licence without any refusal, and the gate that exists to catch a reading exceeding its licence would be enforcing the wrong one.

## Grounds

- pursued: the audit found the locator validated a regime's value but never its agreement with the position, so a drifted definition loaded clean while CI went red, and the gate that exists to catch a reading exceeding its licence would have been enforcing the wrong one; what would show this wrong is a definition resolving under a mismatched position without a refusal.
