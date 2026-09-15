---
schema_version: 1
id: "iss-2609020626111484"
slug: "adr-owed-the-principles-family-becomes-a-declared-record-sto"
severity: "major"
category: "architectural-insight"
source: "agent-finding"
found_during: "iteration-2 planning, decomposition of the cold-reading intents, 2026-09-02"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: ".abcd/development/principles"
resolution: "Adopted by the maintainer on 2026-09-02 as an ADR in this change."
impact: internal
---

ADR owed: the principles family becomes a declared record store with typed frontmatter, a record-architecture decision on the pattern of adr-30, and the lifeboat principles contract moves with it

The record-discipline review of the Iteration 2 intents (2026-09-02) routed this
part out of the knowledge-record intent under the decompose-before-filing rule.
That intent adds four typed keys to principle entries (claim type, reference,
comparison, evidence), a projection rule so a principle's statement reaches a
reading while its citations do not, and a check that a principle resting on a
falsified scope condition is reported.

Today the principles family under the durable record has no frontmatter, no
identifiers, and no entry among the record stores the schema gate walks; its
entries are prose files keyed by filename. Making it a declared store with a
required key set is a record-architecture decision on the pattern of adr-30
(which fixed the record's families and their homes), and it reaches the
lifeboat: `disembark principles` distils into a principles payload whose shape a
packed lifeboat carries, and a consumer of that payload sees the change. The
ADR states that the family is a record store, what keys it carries, that
population is forward-only (an entry carrying no key is untyped, never wrong),
and how the lifeboat's principles schema version moves.

The intent that carries the keys, the projection, the check and the distil
change is the knowledge-record intent filed the same day; it does not ship
until this ADR is adopted.

## Grounds

- pursued: we expect a decision record adopted before the intent that waits on it to keep the trust rule out of the intent's scope list, which the decompose-before-filing rule requires; an intent shipping ahead of its ADR would show it wrong
