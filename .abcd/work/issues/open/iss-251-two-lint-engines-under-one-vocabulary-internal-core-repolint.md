---
schema_version: 1
id: "iss-251"
slug: "two-lint-engines-under-one-vocabulary-internal-core-repolint"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "intent-planning-interview"
found_at: "internal/core/audit"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: consolidating the repolint and lint rule models is a deliberate future intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Two lint engines under one vocabulary: internal/core/repolint (post-itd-6 rename; Rule/Evaluate/tri-state exit) and internal/core/lint (RuleConfig/Finding stream) are separate engines with separate rule models and config schemas, already entangled (repolint's rule_docs.go wraps docs-lint as a conformance rule). Every new rule author must pick an engine. Consolidating the rule models is a deliberate future intent - deliberately kept out of the itd-123-era rename sweep after adversarial review of merge-now vs rename-only.