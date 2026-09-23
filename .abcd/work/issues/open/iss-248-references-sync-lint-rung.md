---
schema_version: 1
id: "iss-248"
slug: "references-sync-lint-rung"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "academic-references-baseline"
found_at: ".abcd/development/research/references.csl.json"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: references_sync lint rung (CSL store vs ACKNOWLEDGEMENTS) is a new rule with no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

references_sync lint rung: cross-check references.csl.json keys/DOIs/URLs against the ACKNOWLEDGEMENTS.md References & sources section, so a store entry without a prose entry (or vice versa) is a finding; today the sync is the documented protocol in research/_references.md