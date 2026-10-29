---
schema_version: 1
id: "iss-2610262033569882"
slug: "the-glossary-allows-bare-the-ledger-only"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 454 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/ledger.md:44"
remedy: "Name 'the sources ledger' where 07-library.md says bare 'the ledger' for the provenance store (line 247 among others), or let the ledger glossary entry allow the bare form inside the library chapter, where the ledger sub-verb lives."
finding_ref: cfk-e24eac9749ba609a
---

The glossary allows bare 'the ledger' only for the issue ledger; the library chapter uses bare 'the ledger' for the sources ledger.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/ledger.md:44` — "Use bare "the ledger" only where the issue ledger is unambiguous — inside `capture` surface"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:246` — "the host records the influence through the ledger and tells the person which key was recorded against which decision"

glossary/core/ledger.md says to use bare 'the ledger' only where the issue ledger is unambiguous and to name every other ledger in full, and lists the sources ledger as one of those others. 07-library says the host 'records the influence through the ledger', meaning the per-repository sources ledger. The chapter uses the bare form for a sense the glossary says must always be named in full.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 454.
