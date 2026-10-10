---
schema_version: 1
id: "iss-2608261437043962"
slug: "itd-5-names-a-phantom-task-classes-enum"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md"
resolution: "itd-5 points task_classes at the naming registry with intent_audit, citing iss-265"
impact: fix
resolved_by:
  commit: "c961ce60"
---

itd-5 names a phantom task_classes enum as source of truth and carries the retired intent_review token