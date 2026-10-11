---
schema_version: 1
id: "iss-2609301307566557"
slug: "abcd-implement-record-transcript-stores"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Carry the capture's scan gap on the run record's transcript entry (scan_gap, home-redacted) and render it under the entry with the same scanGapLines history capture uses; grounds: the gap notice gitleaksAug added for iss-2608291814575788 is the repository's stated disclosure for a masked-by-native-only store, and loopLanding's record verb reaches the same store."
refines: [iss-96]
resolution: "implement record's transcript entry carries the capture's scan gap (scan_gap in --json, a scan gap block in the text), as history capture names it."
impact: fix
resolved_by:
  commit: "f41a1a7b4"
---

abcd implement record --transcript stores each transcript through the history capture but drops the capture's scan gap: in a repository that armed gitleaks where gitleaks is not installed, the run's transcripts are stored with the native scanner only and neither the text render nor --json says so, while history capture names the gap (iss-2608291814575788). The run record's transcript entry carries no field for it.

## Grounds

- pursued: a run transcript stored without the gitleaks coverage the repository armed is disclosed on the run record as history capture discloses it; shown wrong if the record of such a capture carries no scan_gap or the text render omits it
