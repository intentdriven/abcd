---
schema_version: 1
id: "iss-2608270908348796"
slug: "probe-s-record-home-listing-reads-at"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/lifeboat/probe.go"
resolution: "a graveyard listing past the directory cap appends a per-scan truncation notice"
impact: internal
resolved_by:
  commit: "8219937a"
---

probe's record-home listing reads at most maxDirEntries entries per directory, so a home beyond the bound silently drops records from every graveyard scanner's input set with no per-scan notice — the input-set sibling of the resolved per-signal cap announcements, beside the iss-134 determinism note