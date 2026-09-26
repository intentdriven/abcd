---
term: step
bounded_context: core
definition: One of the ordered, independently landable pieces a spec lists under its Steps section; each step is one lane and one pull request, and a spec with no steps is one step.
aliases: []
forbidden_synonyms: ["task", "sub-task", "scope"]
status: stable
introduced_in: adr-2609212115255771
starts_when: null
ends_when: null
not_to_be_confused_with: core/spec
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# step

A **step** is the unit below a spec (itd-2609212103565953): the spec's author lists them in order, each with its own footprint, and `abcd build` lands them one at a time, the next starting after the previous has merged. A step that does not fit a cut leaves the rest as the spec's remainder. The loop's own step interface (`abcd implement step`) uses the same word for the same thing: the next piece of the current spec.

## When to use

When a spec is larger than one implementer can hold and land. Not as a task tracker: a step is a section of the design record, not a record of its own.

## Related terms

- [record families](record-families.md) — the one page that maps the record families and how they relate
