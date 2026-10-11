---
schema_version: 1
id: "iss-2608270655495573"
slug: "memory-persisted-file-write-path-still"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security-cut-agent-flagged-siblings-2026-08-27"
found_at: "internal/core/memory/yaml.go"
resolution: "the memory write path sanitises frontmatter scalars and derived fields while preserving the newline/tab/CR escaping provenance depends on"
impact: fix
resolved_by:
  commit: "cea65342"
---

memory persisted-file WRITE path still emits raw control characters: yaml.go dumpString, RenderIndex/RenderContradictions/renderLogEvent, and the source-block licence written into page frontmatter are not run through termsafe.CleanProse, so a raw cat/git diff/pager over a committed memory page replays escapes. Lower-acuity write-time follow-up to the terminal-render fix (#250/#262), which closed the acute read path. Flagged by the #250/#262 fix agent.