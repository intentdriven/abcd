---
schema_version: 1
id: "iss-2608270945469978"
slug: "gvadridfromfilename-still-parses-the"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
resolution: "gvADRIDFromFilename parses the ordinal textually, so an over-int filename ordinal keeps identity"
impact: internal
resolved_by:
  commit: "8219937a"
---

gvADRIDFromFilename still parses the filename digit run with Atoi and returns empty on an over-int ordinal, so a huge filename ordinal with no frontmatter id skips the record silently — the filename-fallback edge of the resolved textual-canonicalisation fix