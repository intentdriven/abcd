---
schema_version: 1
id: "iss-2608220134344680"
slug: "frontmatter-bom-multiline-comment"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: "internal/core/lint/lint.go"
resolution: "shared frontmatter.TrimBOM plus stateful multi-line-comment skip in the lint and glossary scanners; a BOM- or comment-led record now reaches the blockers"
impact: fix
resolved_by:
  commit: "1f5866e"
---

record-lint frontmatterOpen and frontmatter.Fields anchor on line 0 and TrimSpace does not strip a UTF-8 BOM, so a BOM-led or multi-line-comment-led record yields an empty field map and slips the no_git_metadata blocker and the record_schema id/supersession gates entirely