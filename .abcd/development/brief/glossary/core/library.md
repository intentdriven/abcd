---
term: library
bounded_context: core
definition: The person's collection of outside material (papers, links and the cited notes drawn from them), each item marked confidential or public when it is added, and consulted on request with every answer citing the item it came from.
aliases: []
forbidden_synonyms: ["corpus", "knowledge base"]
status: draft
introduced_in: itd-2610090831227812
starts_when: null
ends_when: null
not_to_be_confused_with: [core/memory-note, core/record-families]
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# library

The **library** is evidence you consult. You add a paper or a link with `/abcd:library add`, and abcd asks whether it is confidential or public when you did not say. Later you ask it a question, "what do my sources say about pacing?", and every answer cites the item it came from. A confidential item can inform a decision but is never named in anything public.

## When to use

For material that comes from outside the project and that a decision may need to cite: a paper, a standard, a vendor's documentation, a page someone sent you.

## When NOT to use

For what an agent learned while working ("this test is flaky under load"): that is a [memory note](memory-note.md). For the project's own decisions: those are records (an ADR, an intent, the decision log).

## Examples

- You add a research paper as confidential; an intent's prior-art section can rest on it, and no public page names it.
- You ask the library what it holds on rate limits, and get three cited passages.

## Related terms

- [memory note](memory-note.md): know-how that finds you, where the library is evidence you consult
- [record families](record-families.md): the project's own records, which a library item may inform but never replaces
