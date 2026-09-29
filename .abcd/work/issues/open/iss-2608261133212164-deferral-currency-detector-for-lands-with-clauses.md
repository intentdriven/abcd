---
schema_version: 1
id: "iss-2608261133212164"
slug: "deferral-currency-detector-for-lands-with-clauses"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "internal/core/lint/contextcurrency.go:20"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: no deferral-currency rule exists (record-lint.json carries context_citation_currency and no sibling), and adding one is a new lint rule with no planned intent. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

a record clause that defers a property pending a named record stays deferred after that record resolves; a deferral-currency sibling of context_citation_currency would catch the invariant-12 class