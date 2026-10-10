---
term: technical-facilitator
bounded_context: core
definition: The person who decides how the work is carried out — who runs the agents and operates the machinery between the product thinker's decisions: the gates, merges, CI, hooks, installs and the mechanics of the record. One of the two people abcd addresses; itd-97 holds that the role is a mode, not a person.
aliases: ["technical facilitator", "facilitator"]
forbidden_synonyms: []
status: stable
introduced_in: itd-2609212137129937
starts_when: null
ends_when: null
not_to_be_confused_with: [core/product-thinker, core/record-families]
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# technical-facilitator

The **technical facilitator** decides *how*: the person at the terminal running the agents, who
operates the gates, merges, CI, hooks and installs, and addresses the mechanism and the record
ids that the [product thinker](product-thinker.md) is spared. A stop that waits on them is
announced as one: a question's `Tech` or `Setup` chip, or `abcd mode facilitator` at a stop
that is not a question, parks the loop on them, and the status line reads `waiting on the
technical facilitator` until they answer.

itd-97 (a draft) holds that **the facilitator is a mode, not a person**. A project runs duo,
with a human technical facilitator beside the product thinker, or solo, where abcd itself
performs the facilitator's work; the gates are the same in both, and only who operates the
machinery between the product thinker's decisions changes.

## When to use

Name the technical facilitator wherever a sentence asks a person to act on the machinery: to
answer an install question, clear a gate, merge, settle where a file lives, or keep a hook they
put in place.

## When NOT to use

Not for what is built, a ruling, an adoption, a dependency sign-off or an irreversible act —
those are the product thinker's. Where a sentence genuinely means either person, it says "the
person" or names both.

## Examples

- "abcd writes what is missing and never replaces what the technical facilitator put there."
- "Ask the technical facilitator first, under a `Tech` chip, then pipe the answer."

## Related terms

- [product-thinker](product-thinker.md), the other person abcd addresses.
- [record-families](record-families.md), the records whose mechanics they operate.
