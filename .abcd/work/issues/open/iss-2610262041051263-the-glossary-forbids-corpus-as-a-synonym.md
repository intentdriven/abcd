---
schema_version: 1
id: "iss-2610262041051263"
slug: "the-glossary-forbids-corpus-as-a-synonym"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 976 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/library.md:6"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Is 'corpus' a forbidden word for the library (glossary/core/library.md, status draft), or is 'sources corpus' the sanctioned name of the library's on-disk store, as the --corpus flag, commands/library.md and the library, banlist and surfaces chapters have it? Answers: (1) the glossary drops 'corpus' from forbidden_synonyms and names 'sources corpus' as the store beneath the library; (2) every chapter, command page and the flag are renamed. Recommend (1): the shipped flag and command pages already use the word, a rename is breaking, and glossary/core/ledger.md already says 'sources corpus'."
related_intents: [itd-77]
finding_ref: cfk-ba11d2eb9606b893
---

The glossary forbids 'corpus' as a synonym for the library; the relocatable-home draft calls the person's sources the 'confidential-sources corpus'.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/library.md:6` — "forbidden_synonyms: ["corpus", "knowledge base"]"
- End B: `.abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17` — "the confidential-sources corpus"

library.md names the person's collection of outside material, marked confidential or public, the library, and lists 'corpus' as a forbidden synonym. itd-77 refers to that same per-person collection as the confidential-sources corpus.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 976.
