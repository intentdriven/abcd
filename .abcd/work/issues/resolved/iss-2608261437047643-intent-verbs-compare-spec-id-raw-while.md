---
schema_version: 1
id: "iss-2608261437047643"
slug: "intent-verbs-compare-spec-id-raw-while"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/intent/lifecycle.go"
resolution: "intent verbs resolve spec_id by number via shared SameNum/HasNum; lint-green spellings verb-green"
impact: fix
resolved_by:
  commit: "c3a46c59"
---

intent verbs compare spec_id raw while lint canonicalises so a lint-green slug or zero-padded spelling bricks reconcile and ready