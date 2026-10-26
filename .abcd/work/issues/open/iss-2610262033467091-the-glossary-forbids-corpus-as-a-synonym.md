---
schema_version: 1
id: "iss-2610262033467091"
slug: "the-glossary-forbids-corpus-as-a-synonym"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 436 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/library.md:6"
remedy: "none (filed automatically)"
finding_ref: cfk-87e633a72b69d531
---

The glossary forbids 'corpus' as a synonym for the library; the library chapter calls the person's collection the 'sources corpus' throughout.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/library.md:6` — "forbidden_synonyms: ["corpus", "knowledge base"]"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:48` — "Six of the verbs act on the sources corpus: creating it, adding a source,"

glossary/core/library.md defines the library as the person's collection of outside material and lists 'corpus' among its forbidden_synonyms. 07-library calls exactly that collection 'the sources corpus' (six verbs 'act on the sources corpus'), and uses 'library' for the command over two stores, the corpus and the in-repo memory store. The record cannot both forbid the word and use it as the canonical name, and the two pages also disagree on what 'library' names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 436.
