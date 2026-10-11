---
id: itd-2610100704270254
slug: an-abcd-run-works-out-how-many-sub
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201925079472]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# An abcd run sets how many sub-agents it keeps alive by the models they run on

## Press Release

> An abcd run works out how many sub-agents it may keep alive from the models they run on. The repository's own config sets the budget, eight of the middle model class by default as abcd ships it, and each class above or below it counts double or half: four of the top class, sixteen of the class below, thirty-two of the smallest, and classes mix by share, so four of the top class and eight of the middle use the whole budget. The class names are vendor-neutral, with each vendor's models mapped onto them; Anthropic's are Fable, Opus, Sonnet and Haiku.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- How does a run know an agent's class? The routing table names a tier per agent (itd-2609201916056194); an agent left at `host-decides` has no class until the host says which model it ran, so the draft must say what it counts as.
- Does the budget replace the pacing intent's `pace.sub_agents` ceiling (itd-2609201925079472, two by default), or sit beside it with the lower of the two winning?
- Does the machine's own config (`~/.abcd.noindex/config.json`), which the pace settings already read between the repository's and the bundled layer, apply here too, or only the repository's config over abcd's default?
- What is this called? The rebuilt pacing lane adds a budget check that compares an estimate of agent runs with a runner's reported quota (abcd-a5, 2026-10-10), so calling this a sub-agent "budget" would give one word two meanings; the glossary entry is settled at planning.

## Decisions

- 2026-10-10: the product thinker filed this as one intent ("File as proposed"), after the facilitator pointed out that their rough rule, "times 2 minus 1", gives 15 of the class below eight but 4.5, not 3, of the class above. Asked whose sub-agents the budget covers, they chose "Each abcd run": only the agents a build or a drain starts count, not a session's hand-started agents or another session's. Asked how the classes trade, they chose "Double per class": each class up counts double and each class down half, four, eight, sixteen and thirty-two from the top, mixed by share. They added that the setting is the repository's, with abcd's bundled default, and is never kept in an agent's memory or anywhere like it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
