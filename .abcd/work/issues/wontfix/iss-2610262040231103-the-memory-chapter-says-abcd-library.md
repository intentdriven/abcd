---
schema_version: 1
id: "iss-2610262040231103"
slug: "the-memory-chapter-says-abcd-library"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 927 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:197"
remedy: "none (filed automatically)"
finding_ref: cfk-ab404d43ce3793f1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033471150. Same End A; add belongs to the sources half of /abcd:library."
duplicates: [iss-2610262033471150]
---

The memory chapter says /abcd:library ships ingest, ask and lint and nothing else; the library glossary entry says items are added with /abcd:library add.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:197` — "The shipped surface carries `ingest`, `ask` and `lint` and nothing else."
- End B: `.abcd/development/brief/glossary/core/library.md:18` — "You add a paper or a link with `/abcd:library add`"

07-memory lists the library verb's shipped surface as exactly ingest, ask and lint and rules out anything else. The library entry says a person adds a paper or a link with /abcd:library add. The add sub-verb cannot be both shipped and excluded.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 927.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033471150. Same End A; add belongs to the sources half of /abcd:library.
