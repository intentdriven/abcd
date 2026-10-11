---
schema_version: 1
id: "iss-227"
slug: "installdevshim-silently-swallows"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "ahoy install dogfood"
found_at: "internal/core/ahoy/apply.go"
resolution: "installDevShim's remove, mkdir and write failures each leave a note naming what was not done; the same silence in the pinned entry's dev-shim removal, the starter settings, rules, setup stamp and machine registration writes is fixed alongside."
impact: fix
resolved_by:
  commit: "f234ae28"
---

installDevShim silently swallows failures: the os.Remove, MkdirAll, and WriteFileAtomic error paths are bare returns (internal/core/ahoy/apply.go, installDevShim), so a failed shim write yields status=partial or clean with no note explaining what was not done or why; only the success path calls a.note

## Grounds

- pursued: every failed install write reaches the result as a note; shown wrong if a forced failure in any of these steps leaves the notes empty, which the dev_shim_failure tests force
