---
schema_version: 1
id: "iss-94"
slug: "the-intent-corpus-still-specifies-the-pre-adr-35-lifeboat-mo"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "itd-88-m0"
found_at: ".abcd/development/intents"
resolution: "The nine intents were re-read at 8322cdf65. itd-24 (the one planned) and the drafts itd-8, itd-9, itd-13, itd-15 and itd-19 spelled the retired 'disembark to <path>' / 'disembark to home' invocation and now read 'disembark pack <repo> <path>', the shape the binary registers. itd-9's acceptance wrote voyage provenance in-tree under .abcd/development/voyage/ and now names the operator-level ~/.abcd/voyage/<source-root-sha>/ store adr-35 decision 3 moved it to; itd-10's question about backing up an in-tree .abcd/lifeboat/ is struck as moot, since a lifeboat is written out of tree. itd-22 carried none of the three already, and itd-2 is superseded and keeps its history. The sweep over every non-superseded intent finds no remaining occurrence except itd-88's audit notes, which cite the old spelling as evidence, and itd-23's proposed to-spec-kit sub-verb, which is a different verb. No gate reads intent prose for retired signatures, so none would have caught this."
impact: internal
resolved_by:
  commit: "7f66911c1"
---

The intent corpus still specifies the pre-adr-35 lifeboat model: itd-2, itd-8, itd-9, itd-10, itd-13, itd-15, itd-19, itd-22 and itd-24 variously use the retired 'disembark to home' signature, the in-tree .abcd/lifeboat/ home, or the in-tree .abcd/development/voyage/ path (itd-9's acceptance writes voyage provenance in-tree — the exact path that would fail abcd's own privacy-hygiene audit rule). The brief, glossary and roadmap were reconciled to adr-35; the intents were deliberately NOT rewritten, because an intent is a proposal with its own lifecycle and silently rewriting nine of them inside an unrelated change is worse than tracking the drift. Each reconciles when it is next planned.

## Grounds

- pursued: no live intent spells the pre-adr-35 lifeboat model; a grep of non-superseded intents for 'disembark to ', '.abcd/lifeboat/' or '.abcd/development/voyage' printing a line outside itd-88's evidence would show it wrong
