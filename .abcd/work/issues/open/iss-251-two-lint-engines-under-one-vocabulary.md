---
schema_version: 1
id: "iss-251"
slug: "two-lint-engines-under-one-vocabulary"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "intent-planning-interview"
found_at: "internal/core/audit"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: should consolidating the repolint and lint rule models become an intent? An architectural insight that proposes new work is not filed without the product thinker's adoption."
remedy: "Waits on ruling (adopt the consolidation as an intent): if adopted, draft an intent that moves internal/core/repolint's rules onto internal/core/lint's RuleConfig and Finding model behind one registry, keeping repolint's tri-state exit as a renderer over findings, proven by every existing rule test passing unchanged against the one engine and a test that each rule id registers once; if declined, wontfix with a rule-author note in internal/README.md saying which engine takes which kind of rule."
---

Two lint engines under one vocabulary: internal/core/repolint (post-itd-6 rename; Rule/Evaluate/tri-state exit) and internal/core/lint (RuleConfig/Finding stream) are separate engines with separate rule models and config schemas, already entangled (repolint's rule_docs.go wraps docs-lint as a conformance rule). Every new rule author must pick an engine. Consolidating the rule models is a deliberate future intent - deliberately kept out of the itd-123-era rename sweep after adversarial review of merge-now vs rename-only.

## Remedy grounds (2026-09-29)

Why: the Finding stream is the richer model and repolint already wraps docs-lint as a rule, so the direction of the merge is set by the entanglement the record names. Rejected: a rename-only sweep, which the record says adversarial review already turned down because it leaves two rule models under one name.
