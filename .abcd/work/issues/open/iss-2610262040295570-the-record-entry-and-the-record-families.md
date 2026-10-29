---
schema_version: 1
id: "iss-2610262040295570"
slug: "the-record-entry-and-the-record-families"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 935 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record.md:32"
remedy: "Reconcile the two uses of 'record family': record.md senses row 3 should name the filed id kinds (adr, itd, iss, spc, dcn and the reading ids) and point to record-families.md; record-families.md should say which of its seven rows are filed families and which (step, bundle, release, status) are groupings or views, and add or exclude ADR and decision rows explicitly."
finding_ref: cfk-3031eb3efc669510
---

The record entry and the record-families page use 'record family' for two different sets.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record.md:32` — "One lifecycle-bucketed set of records of a single kind"
- End B: `.abcd/development/brief/glossary/core/record-families.md:4` — "The one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status)"

The record entry defines a record family as a lifecycle-bucketed set of filed, identified records of one kind, pointing at the four families embark writes (ADRs, issues, intents, specs). The record-families page uses the same phrase for intent, spec, step, bundle, issue, release and status. That list leaves out ADRs and includes step, bundle, release and status, none of which is a filed record with its own id. One name covers two sets.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 935.
