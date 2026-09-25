---
schema_version: 1
id: "iss-265"
slug: "task-classes-crosscheck-claim-unbacked"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "spc-28 build"
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
resolution: "already fixed at tip: the task_classes row in 04-naming.md was reworded (e3964de1) to name the table as the source of truth and to state that no schema or cross-check exists"
impact: internal
resolved_by:
  commit: "e3964de196f6db00b31036e284a73cc0825af318"
---

02-constraints/04-naming.md's task_classes enum row claims 'Machine-readable source of truth: the Go binary's task_classes schema (internal/core/...) — a cross-check test fails if this table and the schema diverge', but no task_classes schema or cross-check test exists anywhere in the Go tree (grep confirms zero Go references). spc-28's step 3 instructed updating that test in the same commit as the intent_review→intent_audit token flip; nothing existed to update. Either build the cross-check the row promises or reword the row to name the real source of truth (agent frontmatter + this table).

## Grounds

- pursued: the naming row makes no claim the tree cannot back; shown wrong if the row again promises a schema or a cross-check test that does not exist
