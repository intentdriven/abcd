---
schema_version: 1
id: "iss-2610262032432036"
slug: "ahoy-calls-the-library-s-store-the-local"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 309 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:472"
remedy: "none (filed automatically)"
finding_ref: cfk-4f9c91725a376b78
---

ahoy calls the library's store 'the local sources corpus'; the library glossary entry forbids 'corpus' as a synonym

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:472` — "the local sources corpus the library verb maintains and /abcd:library reads"
- End B: `.abcd/development/brief/glossary/core/library.md:6` — "forbidden_synonyms: ["corpus", "knowledge base"]"

core/library lists 'corpus' among the forbidden synonyms for the library, the person's collection of outside material. 01-ahoy's inventory describes sources/ as the local sources corpus the library verb maintains and /abcd:library reads, substituting the forbidden word for that same collection.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 309.
