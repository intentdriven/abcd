---
schema_version: 1
id: "iss-2608261437048689"
slug: "firstrootsha-is-the-one-unbounded-git"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/lifeboat/probe.go"
resolution: "firstRootSHA reads through RunLimited with rev-list -n 1; source guard pins the package"
impact: fix
resolved_by:
  commit: "4fe662d4"
---

firstRootSHA is the one unbounded git read on the lifeboat probe path