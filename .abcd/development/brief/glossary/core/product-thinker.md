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
a question's `Product` chip, or `abcd mode product-thinker` at a stop that is not a question,
parks the loop on them, and the status line reads `waiting on the product thinker` until they
answer.

The product thinker owns two things in the record: the brief, which says what the product
is, and the intents, which say what is to be built and in what order. Specs and the issue
ledger are the technical facilitator's to work against those two.

The role does not change with who runs the machinery. itd-97 (a draft) holds that the
facilitator is a mode, not a person — a project runs duo, with a human technical facilitator,
or solo, with abcd doing the facilitator's work — and in both the product thinker's decision
points stay human.

The *product thinker* framing is credited to a [post by signulll](https://x.com/signulll/status/2030404483897815089) (see `ACKNOWLEDGEMENTS.md`, Inspirations).

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
- "Ask the product thinker once, under a `Product` chip, in their own words."

## Related terms

- [technical-facilitator](technical-facilitator.md), the other person abcd addresses.
- [plan](plan.md), the product thinker's sign-off act.
- [record-families](record-families.md), the records whose lifecycle their rulings move.
