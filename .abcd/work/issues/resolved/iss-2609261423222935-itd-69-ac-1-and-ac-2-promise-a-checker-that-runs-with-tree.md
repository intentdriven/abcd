---
schema_version: 1
id: "iss-2609261423222935"
slug: "itd-69-ac-1-and-ac-2-promise-a-checker-that-runs-with-tree"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-69"
origin: researcher-authored
production_mode: hand-written
resolution: "abcd launch manifests --tree public|dev [--root <dir>] runs the manifest lockstep check over a named tree at the chosen polarity through launch.CheckTree, read-only, exiting 0 consistent, 1 drift and 2 unreadable, with no bypass flag; wired on the CLI and the plugin page."
impact: additive
resolved_by:
  commit: "9a33bdcb2"
---

itd-69 ac-1 and ac-2 promise a checker that runs with --tree public and --tree dev. Delivered: launch.CheckLockstep (internal/core/launch/lockstep.go:51) takes the tree as a Go parameter and has no front door — dry-run and ship call it with TreeDev over the source tree (internal/core/launch/dryrun.go:114, ship.go:80) and only the payload render calls it with TreePublic over its own output (internal/core/launch/render.go:444). A public checkout — a marketplace install, a release source archive — cannot be checked for manifest lockstep from the CLI or the plugin surface at all. Wanted: a read-only verb or flag that runs CheckLockstep with a chosen tree against a named root and exits with its 0/1/2 code.

## Grounds

- pursued: a public checkout is checkable for manifest lockstep from the CLI and the plugin surface; shown wrong if launch manifests --tree public over a tree whose marketplace entry disagrees exits other than 1
