---
schema_version: 1
id: "iss-2610262036018615"
slug: "the-read-block-glossary-keeps-every"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 617 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/read-block.md:17"
remedy: "none (filed automatically)"
finding_ref: cfk-6463abb262682173
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031401131. Same omission as iss-2610262031401131, cited against 23-reading.md instead of invariant 15."
duplicates: [iss-2610262031401131]
---

The read-block glossary keeps every reading record of a prior run out of the next run; 23-reading hands the comparative reading a prior widening run's items.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/read-block.md:17` — "The instrument's own outputs, manifests and reading records, sit inside the read-block on the next run."
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:82` — "This is the one positional exception to the prior-run exhaust"

read-block.md states without exception that the instrument's own outputs, manifests and reading records, sit inside the read-block on the next run. 23-reading's comparative position receives the items of one committed widening run, projected to two body fields, and calls this the one positional exception to the prior-run exhaust. The glossary's unconditional statement and the surface's exception cannot both describe the block; a builder reading the glossary would wall off exactly the input the comparative position must receive.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 617.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031401131. Same omission as iss-2610262031401131, cited against 23-reading.md instead of invariant 15.
