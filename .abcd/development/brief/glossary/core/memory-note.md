---
term: memory-note
bounded_context: core
definition: A short lesson an agent learned while working (a fact about the project, the person or the machine), kept with the words that should recall it and brought back into a session automatically when a prompt matches; kept on the person's machine until promoted into the project's record.
aliases: ["memory"]
forbidden_synonyms: []
status: draft
introduced_in: itd-2610091918433290
starts_when: null
ends_when: null
not_to_be_confused_with: [core/library, core/record-families]
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# memory note

A **memory note** is know-how that finds you. An agent writes one with `/abcd:memory add` when it learns something worth keeping ("the person wants every question to carry an example"), together with the words that should bring it back. The same loader that brings in the project's rules brings the note back when a prompt matches those words, so no one has to remember to look it up. A note stays on the person's machine until it is promoted into the project's record, where every agent receives it.

## When to use

For a lesson an agent would otherwise relearn: a preference of the person's, a quirk of the machine, a trap in the project's tooling.

## When NOT to use

For outside material a decision may need to cite: that belongs in the [library](library.md). For a rule every agent in the project must follow: promote the note, so it becomes part of the record.

## Examples

- An agent notes that a test fails under heavy load and should be retried, recalled whenever a prompt mentions that test.
- A note that has been recalled in several sessions is promoted into the project's rules, so a second contributor's agent receives it too.

## Related terms

- [library](library.md): evidence you consult, where a memory note is know-how that finds you
- [record](record.md): where a promoted note ends up
