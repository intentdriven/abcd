---
schema_version: 1
id: "iss-2610262031204475"
slug: "03-mental-model-says-the-step-is-not-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 117 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "Edit glossary/core/record-families.md: map the record families as intent, spec, bundle, issue, release and status, and show step as the `## Steps` section beneath a spec rather than a family row, keeping the record_family_key table and internal/core/lint/recordfamilies_test.go in step."
finding_ref: cfk-e2a234912c17cc11
---

03-mental-model says the step is not a record family; the record-families page lists step as one of abcd's record families.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "The step is a section of the spec, not a record family of its own."
- End B: `.abcd/development/brief/glossary/core/record-families.md:4` — "maps abcd's record families (intent, spec, step, bundle, issue, release, status)"

03-mental-model says 'The step is a section of the spec, not a record family of its own.' and points to the record-families page as the map of every family. That page defines itself as the one page that 'maps abcd's record families (intent, spec, step, bundle, issue, release, status)' and gives step a family row with its own lifecycle. The term record family is drawn to include step in one document and to exclude it in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 117.
