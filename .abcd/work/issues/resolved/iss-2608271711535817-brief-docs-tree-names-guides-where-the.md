---
schema_version: 1
id: "iss-2608271711535817"
slug: "brief-docs-tree-names-guides-where-the"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
resolution: "brief layout tree corrected: guides/->how-to/, assets/ line added"
impact: internal
resolved_by:
  commit: "a032377d"
---

the brief's canonical docs tree drifted from the shipped tree: .abcd/development/brief/05-internals/03-configuration.md line 441 names the task-oriented home guides/ while the shipped directory, docs/README.md and every other reference say how-to/, and the tree block lacks an assets/ line although adr-47 decision 2 sanctions that directory. Correct the tree block — rename guides/ to how-to/ keeping the comment, add the assets/ line citing adr-47. No file moves; the shipped tree is the truth.