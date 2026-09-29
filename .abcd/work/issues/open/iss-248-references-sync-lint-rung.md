---
schema_version: 1
id: "iss-248"
slug: "references-sync-lint-rung"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "academic-references-baseline"
found_at: ".abcd/development/research/references.csl.json"
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: a references_sync rung comparing research/references.csl.json with the references section of ACKNOWLEDGEMENTS.md is a new lint rule with no planned intent. No rule in record-lint.json or the docs lint reads the CSL store, and the sync is still the protocol written in research/_references.md. Plan the rule, or keep the protocol and close. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

references_sync lint rung: cross-check references.csl.json keys/DOIs/URLs against the ACKNOWLEDGEMENTS.md References & sources section, so a store entry without a prose entry (or vice versa) is a finding; today the sync is the documented protocol in research/_references.md