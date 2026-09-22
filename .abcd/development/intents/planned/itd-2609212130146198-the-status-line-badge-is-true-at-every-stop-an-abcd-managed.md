---
id: itd-2609212130146198
slug: the-status-line-badge-is-true-at-every-stop-an-abcd-managed
spec_id: spc-2609212139593041
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-200]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609212137129937, itd-201]
related_adrs: [adr-2609212115255771]
---

# The status-line badge is true at every stop

## Press Release

> **The status-line badge always reads one of three states, a question to the human is refused until the mode is set, and the answer resets it.**
>
> "The badge was set by whichever agent remembered," said a product thinker who had watched it read managed while an agent waited on them. "Now an agent cannot ask me anything until it has said which of us it is asking, and the moment I answer the badge goes back. It is the one signal I have that something is waiting; now it is true."

## Why This Matters

itd-200 shipped the badge and the `mode` verb; the state is whatever the agent last set, so it lags, sticks or never changes, and two captures already sit on it (a default that reads bare "abcd"; a colour that bleeds past the badge). Ruled 2026-09-21: the verb stays the setter, because the agent knows whom it addresses; the guard enforces it; abcd resets it on the next human message.

## Mechanism

We expect a question refused until the mode is set, and a reset on the answer, to make the badge true at every stop, because the two moments the badge must change are the two moments the harness already sees; shown wrong if a stop is still observed with the badge reading managed.

## Scope Conditions

None stated.

## What's In Scope

- **Three states, never bare**: `abcd-managed`, `waiting on the product thinker`, `waiting on the technical facilitator`; an unmanaged repository shows nothing.
- **The guard**: a question to the human through the host's question tool is refused while the mode reads managed; the refusal names the two settings and the verb.
- **The setter**: `abcd mode product-thinker|facilitator` sets the state; the line changes on its next render.
- **The reset**: the prompt hook resets the mode to managed when the next human message arrives after a question, and says so on stderr once.
- **The paint**: the line's colour ends at the badge (closes iss-2609170709035405); the default state closes iss-2609170627427239.

## What's Out of Scope

- Deriving the addressee from the question's text (the agent names it).
- Any state beyond the three.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. The verb sets the state, enforced by the guard on the question tool; the agent is reminded to choose the product thinker or the technical facilitator (ruled 2026-09-21).
2. abcd resets the mode to managed on the next human message.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** an abcd-managed repository, **when** the status line renders, **then** it reads exactly one of `abcd-managed`, `waiting on the product thinker`, `waiting on the technical facilitator`, never a bare `abcd`; an unmanaged repository shows nothing.
- **Given** the mode reads managed, **when** an agent calls the host's question tool, **then** the guard refuses the call naming the two settings and `abcd mode`.
- **Given** `abcd mode product-thinker` or `facilitator`, **when** the line next renders, **then** it shows the corresponding state.
- **Given** a question was asked and the next human message arrives, **when** the prompt hook runs, **then** the mode is reset to managed and one line on stderr says so.
- **Given** the line, **when** it renders, **then** its colour ends at the badge.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the badge is how the product thinker tells an agent is waiting on them, and today it is set by memory; we expect a guarded verb and an automatic reset to make it true at every stop; shown wrong if a stop is still seen with the badge reading managed
