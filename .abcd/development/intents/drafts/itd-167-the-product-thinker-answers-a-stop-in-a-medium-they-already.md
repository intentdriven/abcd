---
id: itd-167
slug: the-product-thinker-answers-a-stop-in-a-medium-they-already
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# The product thinker answers a stop in a medium they already use, and answers it whenever they get to it

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The product thinker is not at a terminal, and requiring them to be there reimposes the constraint this framework exists to remove. A stop that reaches them is answered later, by someone who was not present when it was raised, through a surface they already use.

That makes a stop a durable artefact rather than a prompt: it serialises whole, carries its options with it, outlives the session that raised it, and is resumable by whatever answers it. The loop parks on it and continues elsewhere rather than blocking. The synchronous command-line prompt is the first implementation of that contract, not the contract itself ([adr-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses-and-when-the-loop-st.md)).

Selection over composition is what makes this possible at all: an option set survives a round trip through a web page or an app, and a request for free prose does not.

This does not remove the mediated case, it names it. The command line is the facilitator's surface and a human facilitator works there always, while the product thinker does not work there at all. The two are often in the same room or the same call, with the facilitator reading a question aloud and entering the answer given back. That path stays valid and needs nothing new. What this intent adds is the path where nobody relays: the product thinker answers directly, in their own medium, whenever they get to it, and the loop neither waits at a terminal nor depends on a facilitator being awake.

Whatever the product thinker's surface renders, it renders under the same bounds the register sets: three options at most, a null answer always available, no option marked as preferred, and an explicit statement that no recommendation is being made where the question is a judgement rather than a fact. A surface that quietly reorders options by a score, or that drops the null answer for lack of room, breaks the rule while appearing to keep it.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

**Decided 2026-08-29 (product thinker).** The first surface is a web page the product thinker opens when they have time, with the loop carrying on meanwhile. The mediated path, where a facilitator reads a question aloud, stays valid and is not replaced.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
