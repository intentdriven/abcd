---
term: product-thinker
bounded_context: core
definition: The person who decides what is built and why — who rules on intents, signs off acceptance criteria, adopts or declines a proposal, and owns the decisions no mode automates (adjudication, dependency sign-off, irreversible acts). One of the two people abcd addresses, beside the technical facilitator.
aliases: ["product thinker"]
forbidden_synonyms: []
status: stable
introduced_in: itd-2609212137129937
starts_when: null
ends_when: null
not_to_be_confused_with: [core/technical-facilitator, core/persona, core/record-families]
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# product-thinker

The **product thinker** decides *what*: which intents are pursued, what their acceptance
criteria promise, which proposal is adopted, and every ruling the record carries. `abcd intent
plan <itd-N>` is their sign-off act ([plan](plan.md)), and the planning interview's questions
are theirs. They answer on a surface of their own, so a stop that waits on them is announced:
`abcd mode product-thinker` parks the loop on them, and the status line reads `waiting on the
product thinker` until they answer.

The role does not change with who runs the machinery. itd-97 (a draft) holds that the
facilitator is a mode, not a person — a project runs duo, with a human technical facilitator,
or solo, with abcd doing the facilitator's work — and in both the product thinker's decision
points stay human.

## When to use

Name the product thinker wherever a sentence asks a person to decide what to build, to rule,
to adopt, or to sign off: a question put to them, a stop that waits on them, a ruling cited.

## When NOT to use

Not for how the work is carried out — running the agents, the gates, merges, CI and hooks are
the [technical facilitator](technical-facilitator.md)'s. Not for a [persona](persona.md): a
persona is a modelled archetype in a press release, and its role hint is an outside job title.
Where a sentence genuinely means either person, it says "the person" or names both.

## Examples

- "Adopting a proposal is the product thinker's move."
- "Set `abcd mode product-thinker`, then ask the product thinker once, in their own words."

## Related terms

- [technical-facilitator](technical-facilitator.md), the other person abcd addresses.
- [plan](plan.md), the product thinker's sign-off act.
- [record-families](record-families.md), the records whose lifecycle their rulings move.
