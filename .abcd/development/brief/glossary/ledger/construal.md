---
term: construal
bounded_context: ledger
definition: The statement of what the situation is being treated as, in one or two sentences, held in the brief's framing chapter as the frame a widening reading reads against.
aliases: ["frame statement"]
forbidden_synonyms: ["vision", "hypothesis", "assumption"]
status: draft
introduced_in: itd-183
starts_when: null
ends_when: null
not_to_be_confused_with: core/brief
versions: null
---

# construal

The **construal** is one of the three framing surfaces a reading may see, beside the committed glossary terms and the committed scope (adr-55). It is committed record as it presently stands, and its history, the declined construals and the reasoning that settled them, stays on the local ledger side under adr-55. A passage that describes an intention rather than a commitment opens with the not-yet-real marker.

## When to use

Use it for the frame statement in the framing chapter. A reframe is a change to any of the three framing surfaces, which a reframe record points at without carrying the abandoned text.

## When NOT to use

Do not call the brief's purpose or scope the construal; those are products of the framing, and the construal is the frame itself.

## Related terms

- [brief](../core/brief.md)
- [position](position.md), the widening reading being the one that reads the construal to widen
