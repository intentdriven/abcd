---
schema_version: 1
id: "iss-2610262033471150"
slug: "the-memory-substrate-says-abcd-library-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 438 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:197"
remedy: "Edit 07-memory.md section 9 (Diagnostic paragraph): scope 'ingest, ask and lint and nothing else' to the memory half of /abcd:library and point at 07-library.md for the sources half's verbs."
finding_ref: cfk-2b42c53fea333ce2
---

The memory substrate says abcd library's shipped surface is ingest, ask and lint and nothing else; the library chapter's generated appendix lists nine shipped sub-verbs.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:197` — "The shipped surface carries `ingest`, `ask` and `lint` and nothing else."
- End B: `.abcd/development/brief/04-surfaces/07-library.md:500` — "Sub-verbs: `abcd library add`, `abcd library ask`, `abcd library cite-check`, `abcd library declassify`, `abcd library ingest`, `abcd library init`, `abcd library ledger`, `abcd library lint`, `abcd library sync-banlist`."

07-memory, describing a would-be recall sub-verb of abcd library, says 'The shipped surface carries ingest, ask and lint and nothing else'. 07-library's appendix, generated from the command tree, lists add, ask, cite-check, declassify, ingest, init, ledger, lint and sync-banlist. The 'nothing else' claim cannot hold beside six further shipped verbs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 438.
