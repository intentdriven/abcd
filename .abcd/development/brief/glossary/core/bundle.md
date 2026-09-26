---
term: bundle
bounded_context: core
definition: Several intents that share one spec because they ship as one change; each member carries kind bundle-member and the bundle's name, and all ship together when the spec closes.
aliases: []
forbidden_synonyms: ["phase", "epic", "milestone"]
status: stable
introduced_in: itd-34
starts_when: null
ends_when: null
not_to_be_confused_with: core/step
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# bundle

A **bundle** is a delivery grouping, not a sequencing one: two or three intents whose work is one change, sharing one spec and shipping together when it closes (itd-34, planned, plans them onto that spec in a single act). A bundle says nothing about what comes before what; that is dependencies. A bundle cannot contain its own blocker.

## When to use

When two intents would be one pull request. Not as a phase in disguise: a bundle of ten is a sign the intents were cut wrong.

## Related terms

- [record families](record-families.md) — the one page that maps the record families and how they relate
