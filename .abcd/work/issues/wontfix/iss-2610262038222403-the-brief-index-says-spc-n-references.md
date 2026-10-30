---
schema_version: 1
id: "iss-2610262038222403"
slug: "the-brief-index-says-spc-n-references"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 790 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:8"
remedy: "none (filed automatically)"
finding_ref: cfk-8d0bab2ae41143dd
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035569535. Same README sentence against the status chapter citing spc-2610150728396006 as this repository's design record."
duplicates: [iss-2610262035569535]
---

The brief index says spc-N references trace a predecessor implementation's spec store; the status chapter cites spc-2610150728396006 as this repository's design record, by step.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:8` — "the predecessor implementation's spec store, carried as design input for this"
- End B: `.abcd/development/brief/04-surfaces/39-status.md:4` — "itd-2609151838312703, spc-2610150728396006 step 3, decision 21 of the"

The brief README says `spc-N` references throughout the brief trace the predecessor implementation's spec store, as design input and not as claims about this repository. 39-status cites spc-2610150728396006, with its step 3, as the design record of a verb this repository ships. Both readings of a spc id cannot hold across the brief.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 790.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035569535. Same README sentence against the status chapter citing spc-2610150728396006 as this repository's design record.
