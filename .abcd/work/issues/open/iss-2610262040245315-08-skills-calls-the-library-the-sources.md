---
schema_version: 1
id: "iss-2610262040245315"
slug: "08-skills-calls-the-library-the-sources"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 928 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md:10"
remedy: "none (filed automatically)"
finding_ref: cfk-6e8d2091963365fb
---

08-skills calls the library 'the sources corpus', a word the library glossary entry forbids.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/08-skills.md:10` — "consulting the sources corpus, registering a document in it"
- End B: `.abcd/development/brief/glossary/core/library.md:6` — "forbidden_synonyms: ["corpus", "knowledge base"]"

The library entry lists 'corpus' as a forbidden synonym for the person's collection of outside material. 08-skills calls that same collection, the one consulted and registered into from commands/library.md, 'the sources corpus'.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 928.
