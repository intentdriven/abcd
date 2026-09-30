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
remedy: "Waits on ruling F (plan the rule or keep the protocol): if planned, add a references_sync warn rule to the docs lint that reads research/references.csl.json and the References and sources section of ACKNOWLEDGEMENTS.md and reports a CSL id, DOI or URL present on one side only, entered under the one shared shrink-only warn baseline of ruling BT3, proven by a lint test with one orphan on each side and a clean run on the tree; if the protocol stays, wontfix the record naming research/_references.md as the sync."
---

references_sync lint rung: cross-check references.csl.json keys/DOIs/URLs against the ACKNOWLEDGEMENTS.md References & sources section, so a store entry without a prose entry (or vice versa) is a finding; today the sync is the documented protocol in research/_references.md

## Remedy grounds (2026-09-29)

Why: the two stores are declarable and machine-readable on both sides, so the cross-check is mechanical, and ruling BT3 (2026-09-29) already settles how a new warn rule lands. Rejected: generating the ACKNOWLEDGEMENTS section from the CSL store, which would turn a hand-written credit page into derived output and lose its prose.
