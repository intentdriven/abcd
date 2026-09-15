---
term: lifeboat
bounded_context: core
definition: A portable directory artefact packed by `/abcd:disembark` that captures the distilled knowledge and configuration of a source project so it can be unpacked into a fresh context by `/abcd:embark`. It always lands outside the source repository, at an operator-chosen destination.
aliases: ["lifeboat artefact", "disembark artefact"]
forbidden_synonyms: ["backup", "archive", "snapshot", "checkpoint"]
status: stable
introduced_in: phase-1
starts_when: disembark (`abcd disembark <source-repo> to <dest>`) writes the artefact to the operator-chosen destination; the source repository is never written to.
ends_when: The lifeboat is unpacked by `/abcd:embark` into a target project, or discarded.
not_to_be_confused_with: null
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# lifeboat

A **lifeboat** is the portable artefact that carries a project's accumulated knowledge across
context boundaries. It is produced by `/abcd:disembark` — which reads a source repository without
writing to it, and packs the grounded brief, a coverage report, a graveyard, key decisions,
logbook summary, and configuration into a self-contained directory elsewhere — and consumed by
`/abcd:embark` — which unpacks it to bootstrap a fresh project context.

The analogy: when you leave a sinking ship you take the lifeboat. When a project's context is
full or a rebuild is needed, you disembark with a lifeboat so nothing is lost. And you carry the
lifeboat *off* the ship: the source repository is read-only, so a dead or archived project can be
mined without being modified.

## When to use

Use "lifeboat" when referring to the portable artefact (a directory at an operator-chosen
destination outside the source repository) that enables project-to-project knowledge transfer or
context recovery.

## When NOT to use

Do not use "lifeboat" to mean a generic backup or file snapshot. The term is specifically about
the structured, portable abcd artefact format. Do not confuse with "checkpoint" (which has
flow-control connotations in the abcd pipeline context).

## Examples

- "Run `abcd disembark ../old-project to /tmp/project-lifeboat` to pack a lifeboat before
  starting the rebuild — the old repo is only read."
- "The lifeboat at `/tmp/project-lifeboat/` was passed to `/abcd:embark` to bootstrap the
  new context."

## Related terms

- [voyage](voyage.md) — the lifeboat captures voyage knowledge at a point in time
- [embark](../interview/embark.md) — the grill sub-verb term for the opening move; distinct
  from the `/abcd:embark` command that unpacks a lifeboat
