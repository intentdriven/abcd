---
schema_version: 1
id: "iss-2609261726015043"
slug: "record-lint-s-release-gate-derivation"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
resolution: "changelogAt reads CHANGELOG.md out of git through gitutil.RunCapped at the 4 MiB cap every other CHANGELOG read holds to, and the lint read guard now names an unbounded gitutil.Run asked for a file's content."
impact: fix
resolved_by:
  commit: "9b446292"
---

record-lint's release-gate derivation reads CHANGELOG.md out of a git tree without a bound: changelogAt (internal/core/lint/releasegate_derive.go) runs an unbounded git cat-file blob and holds the whole blob in memory, where every other CHANGELOG read in lint and core/changelog is capped at 4 MiB (maxChangelogBytes, changelog.MaxChangelogBytes). TestLintReadsNothingUnguarded did not see it because its scanner names filesystem opens and reads only (os, io/fs, io/ioutil, an Open/ReadFile method); a file's content read out of git through gitutil.Run passes it. Found sweeping lint for unbounded record reads while integrating review2-match's pre-existing note (readBucket's first read, already guarded at this tip by b48fd584).

## Grounds

- pursued: an oversized CHANGELOG.md blob is refused, fail-closed, and a new unbounded content read out of git in lint fails TestLintReadsNothingUnguarded; a release-gate derivation that parses a blob past 4 MiB, or a gitutil.Run cat-file blob in lint that the test passes, would show it wrong
