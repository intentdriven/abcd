---
term: knowledge-floor
bounded_context: interview
definition: What each of the two people abcd addresses is assumed to know, the measure an explanation in an abcd interview is held to, as against the register, which sets only its tone.
aliases: ["knowledge floor", "floor"]
forbidden_synonyms: ["reading level", "audience level"]
status: stable
introduced_in: itd-201
starts_when: null
ends_when: null
not_to_be_confused_with: [core/persona, core/record-families]
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# knowledge-floor

The register sets the tone; the **knowledge floor** is what an explanation is measured against.
The asking rules every abcd interview follows (the GRILL rule domain) point here in one line,
and this page states the floor in full.

The [product thinker](../core/product-thinker.md) knows the product, its users, what done looks
like, and the ordinary vocabulary of using software: a file, a folder, a name, a version, an
account, a link, a permission someone grants. They are not assumed to know version control (a
checkout, a branch, a merge, a commit, a worktree), a shell, file ownership and permission bits,
continuous integration or a merge queue, a hook, an environment variable, a checksum, a symbolic
link, or the record ids.

The [technical facilitator](../core/technical-facilitator.md) knows all of that as well, so
explaining it to them is padding, and leaving the ids out of their answer withholds the handle
they act on.

A concept below the product thinker's floor that cannot be avoided is introduced in one sentence
in product terms before it is used, without naming the tool that implements it: a lock is one
person holding the pen, a checksum is a fingerprint saying two copies are identical. A question
that fails this is rewritten, not annotated.

## When to use

Use "knowledge floor" when judging whether an explanation or a question suits the person it is
for: what it may take as known, and what it must introduce first. It is the measure behind the
register, which is how the question sounds.

## When NOT to use

Do not use "reading level" or "audience level": the floor is a list of what is known, not a
grade of prose. Do not use it for a [persona](../core/persona.md), a modelled user archetype in
an intent or a brief; the floor belongs to the two people abcd addresses.

## Examples

- "The question named the checkout, which is below the product thinker's knowledge floor; it
  now reads 'the project's folder of work'."
- "For the technical facilitator the record ids stay in: they are above the floor, and leaving
  them out withholds the handle they act on."

## Related terms

- [record families](../core/record-families.md): the one page that maps the record families and how they relate
- [product thinker](../core/product-thinker.md): the person whose floor most explanations are measured against
- [technical facilitator](../core/technical-facilitator.md): the person who knows the mechanism and the ids
- [session](session.md): the interview in which questions are held to the floor
