---
schema_version: 1
id: "iss-2610262032482122"
slug: "disembark-calls-the-packed-lifeboat-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 319 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:280"
remedy: "In 02-disembark.md (the re-pack acceptance near line 280 and the 'versioned pair of snapshot directories' wording) say lifeboat instead of snapshot; glossary/core/voyage.md's 'latest snapshot' wants the same."
finding_ref: cfk-80697634ac786a14
---

disembark calls the packed lifeboat a 'snapshot'; the lifeboat glossary entry forbids 'snapshot' as a synonym

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:280` — "the previous snapshot is replaced and its manifest remains in the voyage log"
- End B: `.abcd/development/brief/glossary/core/lifeboat.md:6` — "forbidden_synonyms: ["backup", "archive", "snapshot", "checkpoint"]"

core/lifeboat lists snapshot among the words that must not substitute for lifeboat. 02-disembark's acceptance says a new snapshot lands at the destination and the previous snapshot is replaced, using the forbidden word for the lifeboat itself.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 319.
