---
schema_version: 1
id: "iss-228"
slug: "the-plugin-root-binary-repo-root-abcd"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "ahoy install dogfood"
found_at: "internal/core/ahoy"
resolution: "Already fixed on main before this lane: 0ead2664 warns at session start when a dogfood plugin-root binary trails its source tip (naming the binary, its revision, the tip and make build), and e7557959 refuses ahoy install through a stale or unknown-vintage binary; the bare ahoy detection reports vintage and staleness."
impact: fix
resolved_by:
  commit: "0ead2664"
---

The plugin-root binary (repo-root abcd -> bin/abcd-darwin-arm64) sat a month stale after iss-171 merged, so the ahoy skill's first resolution rung reported pre-iss-171 gaps (wrong target, missing --bin-dir) with no staleness signal; the skill and detection have no guard that warns when the plugin-root binary predates the source tip in a source checkout

## Grounds

- pursued: a stale plugin-root binary in a source checkout is named at session start and refused at install; shown wrong if staleness_test.go's stale case renders no notice or an install through a stale binary writes
