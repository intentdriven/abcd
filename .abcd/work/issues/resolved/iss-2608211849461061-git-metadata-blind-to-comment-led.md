---
schema_version: 1
id: "iss-2608211849461061"
slug: "git-metadata-blind-to-comment-led"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "bughunt-round-6"
found_at: "internal/core/lint/lint.go:1409"
resolution: "lint-local frontmatterFields slices past a leading comment so the no_git_metadata blocker sees comment-led records"
impact: fix
resolved_by:
  commit: "51ed7e1"
---

frontmatter.Fields requires --- on line 0, so a comment-led record file bypasses the no_git_metadata blocker (13 glossary term files)