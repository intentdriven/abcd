---
term: memory-note
bounded_context: core
definition: A short lesson an agent learned while working (a fact about the project, the person or the machine), kept with the words that should recall it and brought back into a session automatically when a prompt matches; kept on the person's machine unless a person puts it in the project's file.
aliases: ["memory"]
forbidden_synonyms: []
status: stable
introduced_in: itd-2610091918433290
starts_when: null
ends_when: null
not_to_be_confused_with: [core/library, core/record-families]
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# memory note

A **memory note** is know-how that finds you. An agent writes one with `abcd memory add` when it learns something worth keeping ("the person wants every question to carry an example"), together with the words that should bring it back. The same loader that brings in the project's rules brings the note back when a prompt matches those words, so no one has to remember to look it up. A note lives in the machine's `~/.abcd.noindex/memory.json` and is recalled in the repository it was written in, or in every repository when written with `--everywhere`. A note in the project's `.abcd/memory.json` is recalled for every agent working in the project; it gets there only through a person's reviewed change to that file, since no command promotes a note.

## When to use

For a lesson an agent would otherwise relearn: a preference of the person's, a quirk of the machine, a trap in the project's tooling.

## When NOT to use

For outside material a decision may need to cite: that belongs in the [library](library.md). For a rule every agent in the project must follow: put it in the project's `.abcd/rules.json`, or the note in the project's `.abcd/memory.json`, in a reviewed change.

## Examples

- An agent notes that a test fails under heavy load and should be retried, recalled whenever a prompt mentions that test.
- A person reviews the notes with `abcd memory list`, removes one that no longer holds with `abcd memory remove <mem-id>`, and copies one that every contributor's agent should receive into the project's `.abcd/memory.json` in a reviewed change.

## Related terms

- [library](library.md): evidence you consult, where a memory note is know-how that finds you
- [record](record.md): where a note in the project's file ends up, reviewed like any other change
