---
schema_version: 1
id: "iss-2608270908349399"
slug: "update-s-homebrew-cellar-refusal-matches"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/update/update.go"
resolution: "update matches a case-variant brew Cellar path via FoldPath, routing to the brew remedy not the generic refusal"
impact: internal
resolved_by:
  commit: "aca9d57a"
---

update's Homebrew Cellar refusal matches brewCellarPrefixes case-sensitively, so a case-variant Cellar path on macOS falls through to the generic foreign-symlink refusal and the operator gets the wrong remedy text