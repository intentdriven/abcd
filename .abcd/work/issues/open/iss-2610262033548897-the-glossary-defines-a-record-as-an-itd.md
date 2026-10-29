---
schema_version: 1
id: "iss-2610262033548897"
slug: "the-glossary-defines-a-record-as-an-itd"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 448 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record.md:4"
remedy: "In record.md replace the closed list 'itd, spc, adr, iss or rdg' with the id kinds the citation grammar and abcd record-id dispatch admit (itd, spc, adr, iss, dcn, adm, srp, rfm, rdg), or point at that one list."
finding_ref: cfk-9b80a35daf87d651
---

The glossary defines a record as an itd, spc, adr, iss or rdg document; the record dispatcher treats dcn, adm, srp and rfm ids as records and gives rdg no dispatch.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record.md:4` — "One identified, filed document that a command mints and a lint gate reads — an itd-N, spc-N, adr-N, iss-N or rdg-id."
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:83` — "`spc-N`, `adr-N`, `dcn-N`, `adm-N`, `srp-N` or `rfm-N` and reports, read-only, what that"

glossary/core/record.md defines a record as one identified, filed document, 'an itd-N, spc-N, adr-N, iss-N or rdg-id'. 08-abcd's abcd < record-id> takes iss, itd, spc, adr, dcn, adm, srp and rfm ids and reports 'what that record is', and says the reading families have no record dispatch. The two pages give different sets as what 'record' covers, so the term's extent is not fixed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 448.
