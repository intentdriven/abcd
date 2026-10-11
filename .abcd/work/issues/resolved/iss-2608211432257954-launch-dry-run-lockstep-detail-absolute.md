---
schema_version: 1
id: "iss-2608211432257954"
slug: "launch-dry-run-lockstep-detail-absolute"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "internal/core/launch/lockstep.go"
resolution: "Strip the os.PathError path at loadJSON so the lockstep detail is path-free in the dry-run success envelope"
impact: fix
resolved_by:
  commit: "5ebb730"
---

launch --dry-run leaks a cwd/home-rooted absolute path into lockstep.detail and would_refuse_on when a pinned manifest is unreadable, violating the no-absolute-paths-in-machine-output norm; the dry-run is a success envelope so scrubPaths never reaches it