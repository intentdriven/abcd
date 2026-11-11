---
schema_version: 1
id: "iss-2611070737163284"
slug: "the-drive-test-worlds-home-directory"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive_test.go"
remedy: "Name the world's home by a fixed or sanitised label, not the subtest name."
resolution: "A drive test world's home is now a directory made under the fixed label abcd-drive-home-, not t.TempDir, so a comma in a subtest's name no longer reaches the walled directory's path and changes what the subtest exercises."
impact: internal
resolved_by:
  commit: "3f7d7ec3f854f90bf3a914bea022afc742f6044e"
---

The drive test worlds' home directory carries the subtest's name, so a comma in a later subtest name silently changes what it exercises; one subtest of the panic-before-first-move test passed that way.

## Grounds

- pursued: a drive test whose subtest name holds a comma starts its drive as one named without does, which TestADriveWorldsHomeDoesNotCarryTheSubtestsName holds; a drive test refused with 'a path a harness deny rule cannot hold' under such a name would show it wrong.
