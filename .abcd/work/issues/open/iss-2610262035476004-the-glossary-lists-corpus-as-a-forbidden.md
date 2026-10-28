---
schema_version: 1
id: "iss-2610262035476004"
slug: "the-glossary-lists-corpus-as-a-forbidden"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 598 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/library.md:6"
remedy: "none (filed automatically)"
finding_ref: cfk-ce9bb6f857b48e7e
---

The glossary lists 'corpus' as a forbidden synonym for the library; the banlist chapter calls the library 'the sources corpus'

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/library.md:6` — "forbidden_synonyms: ["corpus", "knowledge base"]"
- End B: `.abcd/development/brief/04-surfaces/20-banlist.md:129` — "The sources corpus derives patterns from its confidential entries"

library.md defines the library as the person's collection of outside material, each item marked confidential or public, and lists 'corpus' among its forbidden synonyms. The banlist chapter describes that same collection by its confidential entries and points to the library chapter, but names it 'the sources corpus', 'the corpus sync' and 'the corpus writer'. That is the forbidden substitution.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 598.
