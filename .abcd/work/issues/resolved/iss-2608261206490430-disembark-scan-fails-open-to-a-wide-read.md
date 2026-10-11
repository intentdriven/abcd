---
schema_version: 1
id: "iss-2608261206490430"
slug: "disembark-scan-fails-open-to-a-wide-read"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "bughunt-a/round-8"
found_at: "internal/core/lifeboat/probe.go"
resolution: "probe.go parts an empty ls-files listing from a git failure; an all-ignored tree now narrows to nothing. Watched-fail test added."
impact: fix
resolved_by:
  commit: "640c1517"
---

disembark scan fails open to a wide read on an all-ignored repository