---
schema_version: 1
id: "iss-2610262037095269"
slug: "the-record-families-page-lists-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 702 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-49b4bb784f7496f2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040295570. Same root: the record-families page lists seven rows that are not the filed id kinds other documents mean by 'record family' (adr, itd, iss, spc, dcn and the reading ids). One rework of the page and record.md clears it."
duplicates: [iss-2610262040295570]
---

The record-families page lists the record families as intent, spec, step, bundle, issue, release and status; the inbox chapter calls adr, itd, iss, spc and dcn the record families.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:4` — "The one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status)"
- End B: `.abcd/development/brief/04-surfaces/30-inbox.md:108` — "a record family abcd resolves (`adr`, `itd`, `iss`, `spc` and `dcn`) followed by a number"

record-families.md is the one page that maps the record families and lists intent, spec, step, bundle, issue, release and status, with no ADR or decision family. 30-inbox.md uses "record family" for the id prefixes the citation grammar resolves, adr, itd, iss, spc and dcn, two of which have no row on that page. The term names lifecycle groupings in one document and id prefixes in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 702.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040295570. Same root: the record-families page lists seven rows that are not the filed id kinds other documents mean by 'record family' (adr, itd, iss, spc, dcn and the reading ids). One rework of the page and record.md clears it.
