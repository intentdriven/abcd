---
schema_version: 1
id: "iss-2608211432258689"
slug: "launch-dry-run-wouldrefuseon-unsanitised"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "internal/surface/cli/cli.go"
resolution: "Route each launch dry-run refusal reason through termsafe.Sanitize"
impact: fix
resolved_by:
  commit: "5ebb730"
---

launch --dry-run prints WouldRefuseOn with %v and no termsafe.Sanitize, so a committed repo filename carrying raw terminal escapes reaches the terminal and CI log (sibling of iss-259/iss-264 at an unrecorded site)