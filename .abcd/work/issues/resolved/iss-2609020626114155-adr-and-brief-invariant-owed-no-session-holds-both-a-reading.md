---
schema_version: 1
id: "iss-2609020626114155"
slug: "adr-and-brief-invariant-owed-no-session-holds-both-a-reading"
severity: "major"
category: "architectural-insight"
source: "agent-finding"
found_during: "iteration-2 planning, decomposition of the cold-reading intents, 2026-09-02"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "agents/scribe.md"
resolution: "Adopted by the maintainer on 2026-09-02 as an ADR in this change."
impact: internal
---

ADR and brief invariant owed: no session holds both a reading and the ledger, and the session-kind stamp changes the reading bundle's shape

The record-discipline review of the Iteration 2 intents (2026-09-02) routed two
parts out of the scribe-verb intent under the decompose-before-filing rule.

The first is a trust-boundary rule: no session holds both a reading's assembled
input and ledger content. Brief invariant 15 already states the sentence, and
itd-188 states it in the scribe definition as the inverse of the read block.
What the record lacks is the mechanism: an ADR stating why the two contexts
are kept apart (a context holding both is the meeting the whole arrangement
exists to prevent) and what holds it (a per-run context stamp on every
assembly, and a check over the transcript store that reports a transcript
carrying both stamps of one run), and the invariant's amendment naming that
stamp and that check.

The second is plumbing: the stamp changes the reading bundle's shape, which
moves the assembler version on itd-199's precedent, and the brief's reading
surface chapter describes the bundle. Both are the brief's to record.

The intent that carries the two verbs and the check is the scribe-verb intent
filed the same day.

## Grounds

- pursued: we expect a decision record adopted before the intent that waits on it to keep the trust rule out of the intent's scope list, which the decompose-before-filing rule requires; an intent shipping ahead of its ADR would show it wrong
