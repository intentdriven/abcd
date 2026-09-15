---
term: disposition
bounded_context: ledger
definition: The researcher's recorded response to one reading item, written as a separate record keyed to the item, in one of four states: accepted, rejected, declined or held.
aliases: []
forbidden_synonyms: ["resolution", "triage"]
status: draft
introduced_in: itd-183
starts_when: null
ends_when: null
not_to_be_confused_with: ledger/admission
versions: null
---

# disposition

A **disposition** is a second act and a second write. The reading's item exists before it is dispositioned, so the record can always show that a finding existed before it was answered. Rejected names the purpose an intentional constraint protects; held carries an exit condition; accepted and declined carry grounds. Which states are available varies by position, and an item with no disposition is outstanding, which is a report and never a state.

## When to use

Use it for the record and the act of answering a reading item, and for the scope-condition disposition, the same word applied to an ex-ante assumption: survived, narrowed, falsified or untested.

## When NOT to use

Do not call a disposition a resolution: resolved means fixed, and an accepted tension may be accepted and deliberately not acted on. Do not reuse the issue ledger's open, resolved and wontfix.

## Related terms

- [admission](admission.md)
- [regime](regime.md)
