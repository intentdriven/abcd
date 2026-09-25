---
schema_version: 1
id: "iss-248"
slug: "references-sync-lint-rung"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "academic-references-baseline"
found_at: ".abcd/development/research/references.csl.json"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Add a references_sync lint rung comparing the CSL store with ACKNOWLEDGEMENTS as a new rule?"
---

references_sync lint rung: cross-check references.csl.json keys/DOIs/URLs against the ACKNOWLEDGEMENTS.md References & sources section, so a store entry without a prose entry (or vice versa) is a finding; today the sync is the documented protocol in research/_references.md