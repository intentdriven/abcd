---
schema_version: 1
id: "iss-2610262032245453"
slug: "library-says-embark-carrying-the-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 263 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:427"
remedy: "In 07-library.md, 'Composition with adjacent surfaces' (line 427), replace 'carrying the store forward is designed behaviour' with a statement that embark does not carry the memory store (it is not an embark family, 03-embark.md:126); keep the source-class note only if a carry is ever designed."
finding_ref: cfk-28499d3e8ddd6238
---

Library says embark carrying the memory store forward is designed behaviour; embark says the memory store is not an embark family and does not travel

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:427` — "**`/abcd:embark`** carrying the store forward is designed behaviour"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:126` — "Terminology, docs and the memory store are **not** embark families; they do not travel."

Library's composition note says embark carrying the store forward is designed behaviour, with the receiver running the store lint after unpacking. Embark says terminology, docs and the memory store are not embark families and do not travel, and its design-target section lists no memory carry. A builder cannot implement embark both to carry the store and to leave it behind.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 263.
