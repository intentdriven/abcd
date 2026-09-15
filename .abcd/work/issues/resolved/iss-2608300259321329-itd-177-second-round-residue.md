---
schema_version: 1
id: "iss-2608300259321329"
slug: "itd-177-second-round-residue"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-177 second-round reviews, 2026-08-30"
found_at: "internal/core/intent/ready.go (scopeConditionsCheck), internal/core/intent/claims.go (malformedMarkerRe), internal/core/intent/lifecycle.go (stampPlanned), commands/intent.md"
resolution: "The structural faults are judged before the claim-state switch, so a first section reading None stated. can no longer hide a second heading carrying real bullets; the near-miss guard is case-insensitive; a marker quoted in backticks is reported as malformed rather than read as an identity nobody minted; and the stamp refuses before writing a record past the byte cap its own reader enforces, which would otherwise make the next Load refuse the whole corpus. The plugin page's refused-versus-skipped wording was corrected in the preceding commit, where the same sentence had to change for comment spans."
impact: internal
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

itd-177 second-round residue: the structural faults (duplicated heading, fenced section) are checked only after the claim-state switch, so a nullity or empty first section hides a second Scope Conditions heading carrying real bullets and the gate passes while the stamp refuses; the malformed-marker guard is case-sensitive so a capitalised near-miss gets a real marker glued beside it; no size check after stamping, so a record near the read cap can be written larger than its own reader accepts and the next Load refuses the whole corpus; a marker inside an inline code span is accepted and excised; the plugin page says every fault is refused by the stamp where two-marker and duplicated-id bullets are skipped rather than refused.
