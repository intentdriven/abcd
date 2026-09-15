---
schema_version: 1
id: "iss-2609020626100041"
slug: "adr-owed-the-comparative-channel-admits-two-fields-of-one-wi"
severity: "major"
category: "architectural-insight"
source: "agent-finding"
found_during: "iteration-2 planning, decomposition of the cold-reading intents, 2026-09-02"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/reading/include.go"
resolution: "Adopted by the maintainer on 2026-09-02 as an ADR in this change, with the operand this record proposed withdrawn the same day: the ADR derives the widening run from the record and the invocation stays position and target state; the exception to the exhaust and the preset refusal's withdrawal stand as proposed."
impact: internal
---

ADR owed: the comparative channel admits two fields of one widening run's items from the readings family at one position, a positional exception to the prior-run exhaust

The record-discipline review of the Iteration 2 intents (2026-09-02) routed this
part out of the comparative-channel intent under the decompose-before-filing
rule: a positional exception to a read-block invariant is a trust-boundary rule,
and a trust-boundary rule lives in an ADR with a brief invariant, not in an
intent's scope list.

What the exception is. The include table's exclusions state that the
instrument's own output is never its input, and itd-186's fourth criterion pins
it: prior manifests and reading records never reach a reading. The cold-reading
rulings of 2026-08-28 also fix the comparative reading's object as the widening
reading's output before admission, and admit no other source. Both cannot hold
without a stated exception: at the comparative position only, the two body
fields of one named widening run's items (the configuration and what admits it)
plus the item identifier are admitted, and everything else in the readings
family stays excluded. The ADR states the exception, its ground (the candidate
text is cold because the widening reading produced it without ledger access;
the candidates' fate is warm), and its limit (one run, two fields, one
position), and the brief's invariant on the read block is amended to name it.

The same ruling withdraws itd-199's refusal of a preset naming a comparative
scope, since the comparative preset names the criteria discipline the reading
characterises against; and it supersedes adr-58's "to no other" by one more
closed operand, the candidate run, amending brief invariant 15's enumeration
of the operands while leaving its binding property, that no operand carries
prose, untouched.

The intent that carries the operand, the projection and the refusals is the
comparative-channel intent filed the same day; it does not ship until this ADR
is adopted.

## Grounds

- pursued: we expect a decision record adopted before the intent that waits on it to keep the trust rule out of the intent's scope list, which the decompose-before-filing rule requires; an intent shipping ahead of its ADR would show it wrong
