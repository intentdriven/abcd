---
schema_version: 1
id: "iss-2608261338035835"
slug: "record-lands-with-the-act-promotion-never-fired"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "concept-audit"
found_at: ".abcd/development/principles/the-record-lands-with-the-act.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Promote the-record-lands-with-the-act to a discipline intent?"
---

the-record-lands-with-the-act names an armed rung in present tense and its promotion never fired. The principle's Promotion paragraph states 'The armed rung is make lint-issues, a CI step and a pre-push gate' — verified true: RS001-RS003 live in scripts/check-issue-resolution.sh, wired at Makefile lint-issues and included in preflight. Per the principles README promotion contract (enforced principle -> discipline-kind intent), the gate shipping should have minted a discipline; none exists. Third instance of the promotion-pipeline class alongside iss-390 (examples-use-reserved-identifiers) and iss-2608261041210476 (spec-moves-with-the-surface), and the only one with no ledger record until now. The class evidence is now three: the promotion ritual that only a human can perform is the step that gets skipped, which is the-record-lands-with-the-act applied to its own pipeline.