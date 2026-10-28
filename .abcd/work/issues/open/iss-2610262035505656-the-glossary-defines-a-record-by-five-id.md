---
schema_version: 1
id: "iss-2610262035505656"
slug: "the-glossary-defines-a-record-by-five-id"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 601 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-951dfab1621502d3
---

The glossary defines a record by five id kinds and leaves out dcn-; the ideate chapter calls a decision a record cited by its dcn- id like any other

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record.md:4` — "an itd-N, spc-N, adr-N, iss-N or rdg-id"
- End B: `.abcd/development/brief/04-surfaces/18-ideate.md:57` — "A decision record is cited by its `dcn-` id like any other record."

record.md's definition lists what a record is: an itd-N, spc-N, adr-N, iss-N or rdg-id. The ideate chapter says 'A decision record is cited by its `dcn-` id like any other record' and holds leg 2's citations to records alone. By the glossary's definition a dcn- decision is not a record, so it could not be cited as 'any other record' is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 601.
