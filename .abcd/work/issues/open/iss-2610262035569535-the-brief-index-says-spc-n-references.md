---
schema_version: 1
id: "iss-2610262035569535"
slug: "the-brief-index-says-spc-n-references"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 610 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:8"
remedy: "none (filed automatically)"
finding_ref: cfk-b93e13904d001792
---

The brief index says spc-N references trace the predecessor implementation's spec store and are not delivery claims; the update chapter cites spc-32 as a closed spec in this repository's own store, paired with a shipped intent

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:8` — "the predecessor implementation's spec store, carried as design input for this"
- End B: `.abcd/development/brief/04-surfaces/21-update.md:252` — "[spc-32] (../../specs/closed/spc-32-abcd-update-completes-a-chosen-update-in.md)"

The brief README says `spc-N` references throughout the brief trace the predecessor implementation's spec store, are carried as design input for the Go rebuild, and are not delivery claims. The update chapter's references give 'Intent / spec' as shipped itd-130 and spc-32 at this repository's `specs/closed/` path, so spc-32 is this repository's delivered spec, not the predecessor's. Other chapters cite timestamp-minted ids such as spc-2610030944505997 the same way.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 610.
