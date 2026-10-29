---
schema_version: 1
id: "iss-2610262038235036"
slug: "the-brief-index-says-delivery-state"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 791 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:13"
remedy: "none (filed automatically)"
finding_ref: cfk-38c9921f1fed83fb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035572805. Same A end; the other end is 04-surfaces/README.md's machine-checked Status column instead of 10-docs.md's table."
duplicates: [iss-2610262035572805]
---

The brief index says delivery state never lives in brief prose; the surfaces register says 06-delivery, a brief chapter, carries it in detail.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:13` — "lives in the roadmap dashboard, never in brief prose (per"
- End B: `.abcd/development/brief/04-surfaces/README.md:10` — "carries the delivery state in detail"

The brief README says current delivery state lives in the roadmap dashboard, never in brief prose (adr-5). 04-surfaces/README keeps a machine-checked shipped/staged Status column and says `06-delivery/` carries the delivery state in detail. Both are brief pages, so the brief either holds delivery state or never does.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 791.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035572805. Same A end; the other end is 04-surfaces/README.md's machine-checked Status column instead of 10-docs.md's table.
