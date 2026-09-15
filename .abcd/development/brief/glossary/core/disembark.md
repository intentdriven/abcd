---
term: disembark
bounded_context: core
definition: The act of packing a lifeboat — `abcd disembark <source-repo> to <dest>` reads a source repository without writing to it and distils its settled artefacts, decisions, and configuration into a portable lifeboat directory at a destination outside that repository, which a fresh context can later unpack via `/abcd:embark`.
aliases: ["lifeboat packing", "disembarkation"]
forbidden_synonyms: ["export", "backup", "dump", "snapshot"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: null
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# disembark

**Disembark** is the surface that packs a [lifeboat](lifeboat.md): a portable, highest-fidelity
proxy of a project's theory that can be carried across a session, machine, or team boundary.

It takes the source repository as an argument and is **read-only and out-of-tree**:
`abcd disembark <source-repo> to <dest>` reads the repository — any repository, including a dead
or archived one abcd has never touched — and writes the lifeboat somewhere else. The source tree
is never written to, so there is no in-tree lifeboat directory; the record of the run lands in the
[voyage](voyage.md) log at the operator level instead. The destination is guarded by a safety
gate: disembark refuses unless it is absent, an empty directory, or one carrying a parseable
`_provenance.json` — it never overwrites a directory abcd did not produce
([adr-35](../../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md), which supersedes
adr-4).

`abcd disembark probe <source-repo>` inspects without writing a lifeboat, reporting **coverage**:
which brief sections the repository can ground, which come back blank, and what was searched. A
blank is a first-class result, not a failure. `dry-run` likewise inspects without writing.

## When to use

Use "disembark" when describing the act of packing a lifeboat — the outbound half of the
lifeboat lifecycle. It is the counterpart to the [`/abcd:embark`](../../04-surfaces/03-embark.md)
unpack surface's inbound opening; together they bracket the portability boundary. (Do not
confuse this with the interview-context [embark](../interview/embark.md), the opening move of
a grill session — a distinct term in a different bounded context.)

## When NOT to use

Do not use "export", "backup", or "snapshot" — these lose the navigational, theory-preserving
connotation. Disembark packs the *floor* the project can carry forward, not a byte-for-byte
copy.

## Examples

- "The disembark pass distilled the shipped intents and decision timeline into the lifeboat."
- "A `disembark probe` lists the sources that would be packed, and the sections it could not
  ground, without writing anything."
- "We disembarked the archived repo to `~/lifeboats/atlas/` — the source was never touched."

## Related terms

- [lifeboat](lifeboat.md) — the artefact disembark packs, at a destination outside the source repo
- [voyage](voyage.md) — the operations namespace at `~/.abcd/voyage/<source-root-sha>/` that
  records each disembark run
