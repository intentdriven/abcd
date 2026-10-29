---
schema_version: 1
id: "iss-2610262031070645"
slug: "the-context-page-calls-the-library-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 88 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:13"
remedy: "In the library-versus-memory paragraph of 04-surfaces/README.md, say the library's page store (the 'memory store' at .abcd/memory/, the name commands/library.md and 07-library.md use) is distinct from the memory notes the /abcd:memory command keeps, so 'memory store' and 'memory notes' are not read as one thing."
finding_ref: cfk-2592e631a03ee510
---

The context page calls the library's curated pages a 'memory store', but the surfaces index gives the name memory to the notes command only

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:13` — "curate a queryable memory store (`ingest` external sources / `ask` queries / `lint` health-checks)"
- End B: `.abcd/development/brief/04-surfaces/README.md:494` — "That merge freed the name `memory`, which now belongs to the notes command"

The context page describes `/abcd:library` as curating a queryable memory store, with ingest/ask/lint over specs, ADRs, reviews and memory. The surfaces index says the library merge freed the name `memory`, which now belongs to the notes command. Its library-versus-memory table puts the cited notes drawn from sources in the library and short agent lessons in memory, and it says the two are not one store with two names. So 'memory store' means the library's curated pages on one page and the notes store on the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 88.
