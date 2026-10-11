---
schema_version: 1
id: "iss-2608211432389477"
slug: "memory-md-source-class-should-be-source"
severity: "nitpick"
category: "documentation"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "commands/memory.md"
resolution: "Correct the citation field name to source_class in commands/memory.md"
impact: fix
resolved_by:
  commit: "1c3ae21"
---

commands/memory.md writes the citation field as source.class (a nonexistent nested path) though the JSON key is the flat source_class