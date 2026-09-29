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
remedy: "Waits on planning F (a new lint rule): if planned, add a deferral_currency rule beside context_citation_currency in internal/core/lint that reads each open record's blocked_by and the record handles its deferral_reason names as pending, and reports one whose named record sits in a terminal folder, proven by a test with a deferral pending a resolved issue (reported) and one pending an open issue (clean); if declined, wontfix naming blocked_by and the drain's blocked rule as the checked form."
---

a record clause that defers a property pending a named record stays deferred after that record resolves; a deferral-currency sibling of context_citation_currency would catch the invariant-12 class

## Remedy grounds (2026-09-29)

Why: context_citation_currency already resolves a handle to its folder, so the sibling reuses that resolver and only changes which text it reads. Rejected: parsing free prose for any deferral clause, which the citation-currency sweep in iss-2609091820182387 measured as mostly false alarms; the rule reads the named fields only.
