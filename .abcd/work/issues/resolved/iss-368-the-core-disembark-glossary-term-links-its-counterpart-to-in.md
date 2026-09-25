---
schema_version: 1
id: "iss-368"
slug: "the-core-disembark-glossary-term-links-its-counterpart-to-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/brief/glossary/core/disembark.md"
resolution: "Repointed the core/disembark glossary counterpart link to the /abcd:embark unpack surface chapter and added a not-to-be-confused-with note for the interview-context embark term."
impact: internal
---

The core/disembark glossary term links its counterpart to interview/embark (the grill-session opening, a different bounded context) rather than the /abcd:embark unpack surface, misdirecting the reader
## Evidence

- `.abcd/development/brief/glossary/core/disembark.md:37-38` — "counterpart to `[embark](../interview/embark.md)`'s inbound opening; together they bracket the portability boundary".
- `interview/embark.md:5` (`bounded_context: interview`) defines embark as "the opening move of a grill session" — a different bounded context.
- disembark's true inbound counterpart is `/abcd:embark` unpack (`04-surfaces/03-embark.md`), which has no `core/` glossary term (`find glossary -iname 'embark*'` → only `interview/embark.md`).

## Adversarial verdict

CONFIRMED (nitpick). Not a dangling link (target exists) but a cross-context misdirection. Fix: repoint disembark.md:38 to the `/abcd:embark` unpack surface chapter, or reword to name the unpack command (no core glossary term).
