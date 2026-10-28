---
schema_version: 1
id: "iss-2610262030472547"
slug: "the-press-release-says-the-library"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 45 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-eb87c6b5c4810775
---

The press release says the library curates its knowledge substrate from the documents you consult; the library chapter says nothing connects the corpus to the memory store

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "`/abcd:library` keeps the documents you consult and curates a queryable knowledge substrate from those same sources"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:8` — "nothing distils a corpus document into the memory store, and nothing moves a memory page into the corpus"

01-press-release says /abcd:library keeps the documents you consult and curates a queryable knowledge substrate from those same sources, while 07-library rules (decision 10) that nothing distils a corpus document into the memory store and nothing moves a memory page into the corpus. The substrate cannot be curated from the corpus documents if the two stores never connect.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 45.
