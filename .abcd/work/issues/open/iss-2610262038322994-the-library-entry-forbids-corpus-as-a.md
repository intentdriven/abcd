---
schema_version: 1
id: "iss-2610262038322994"
slug: "the-library-entry-forbids-corpus-as-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 802 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/library.md:6"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Is 'corpus' a forbidden word for the library (glossary/core/library.md, status draft), or is 'sources corpus' the sanctioned name of the library's on-disk store, as the --corpus flag, commands/library.md and the library, banlist and surfaces chapters have it? Answers: (1) the glossary drops 'corpus' from forbidden_synonyms and names 'sources corpus' as the store beneath the library; (2) every chapter, command page and the flag are renamed. Recommend (1): the shipped flag and command pages already use the word, a rename is breaking, and glossary/core/ledger.md already says 'sources corpus'."
finding_ref: cfk-a5b9f88a20151139
---

The library entry forbids "corpus" as a synonym; the surfaces register says "the corpus" and "the corpus verbs" for the library.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/library.md:6` — "forbidden_synonyms: ["corpus", "knowledge base"]"
- End B: `.abcd/development/brief/04-surfaces/README.md:343` — "run in the host agent around the corpus verbs"

library.md lists "corpus" among the words that must not stand for the library. 04-surfaces/README describes the library page's steps as searching the corpus and running around the corpus verbs, and speaks of consulting the sources corpus, which is the substitution the entry forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 802.
