---
term: voyage
bounded_context: core
definition: The operations namespace at `~/.abcd/voyage/<source-root-sha>/` — an append-only record of what abcd *did* to produce a lifeboat (every disembark and embark run), as against the lifeboat itself, which is what gets carried.
aliases: ["voyage log", "voyage namespace"]
forbidden_synonyms: ["lifeboat", "project", "sprint"]
status: stable
introduced_in: phase-1
starts_when: The first `abcd disembark` against a source repository creates `~/.abcd/voyage/<source-root-sha>/`, keyed on that repository's root-commit SHA.
ends_when: Never by abcd's hand — the log is append-only, outlives any single lifeboat, and is removed only if the operator deletes the directory.
not_to_be_confused_with: core/lifeboat
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# voyage

A **voyage** is the *operations* side of the lifeboat surface: the record of what abcd did, run
by run. It is the verb to the [lifeboat](lifeboat.md)'s noun — the lifeboat is **what gets
carried**, the voyage is **what we did to produce it**. That distinction is load-bearing and is
registered as such ([adr-35](../../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md),
which supersedes adr-4).

A voyage lives at the **operator level**, never inside the source repository and therefore never
committed:

```
~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl   # one line per disembark run
~/.abcd/voyage/<source-root-sha>/embark/provenance.json
~/.abcd/voyage/<source-root-sha>/embark/from/<timestamp>/
```

It is keyed on the source repository's root-commit SHA, matching the convention the history store
already uses. It sits outside the tree because a voyage records absolute source paths, and abcd's
own `privacy-hygiene` audit rule flags those in committed files — an in-tree voyage would have
made abcd fail its own audit. It also keeps [`disembark`](disembark.md) honest: the source repo is
read-only, so the only place a run *can* be recorded is here.

The log accumulates; the lifeboat does not. A lifeboat is the latest snapshot, regenerated in
place — there is never a `lifeboat-v1/` beside a `lifeboat-v2/`. The history of how it came to
look that way lives in the voyage log.

> **Open question (adr-35):** adr-35 registers `voyage/` as the operations namespace only. It does
> not settle whether abcd still needs a term for the *end-to-end project lifecycle* (brief approval
> through final release) — the sense this entry previously carried, and the sense
> [`oracle`](oracle.md) ("artefacts produced during a voyage") and [`phase`](phase.md) ("a full
> lifecycle arc") still lean on. Either a replacement term is coined for that arc, or those entries
> are re-authored to drop it. Naming it is a product decision and is not made here.

## When to use

Use "voyage" for the operations namespace and its contents: the append-only log of disembark and
embark runs against a given source repository, and the provenance those runs leave behind. "The
voyage log says this lifeboat was packed from root SHA `3f9a…` with the host oracle."

## When NOT to use

Do not use "voyage" for the lifeboat itself — the artefact is the lifeboat, and conflating them
loses the noun/verb split the two names exist to make. Do not use it for a "project" (too generic)
or a "sprint" (a time box, not a record of runs).

## Lifecycle

| Phase | Condition |
|-------|-----------|
| Starts when | The first `abcd disembark <source-repo> to <dest>` creates `~/.abcd/voyage/<source-root-sha>/` |
| Ends when | Only if the operator deletes it — the log is appended to, never rewritten or truncated |

## Examples

- "Each disembark run appends one line to the voyage log: `manifest_sha256`, the file list, the
  oracle backend used, and the verdict."
- "The voyage is keyed on the root-commit SHA, so two clones of the same repository share one
  voyage."

## Related terms

- [lifeboat](lifeboat.md) — the artefact (noun); a voyage is the record of producing it (verb)
- [disembark](disembark.md) — the read-only, out-of-tree run that appends to the voyage log
- [brief](brief.md) — the structure a disembark run grounds section by section; what it cannot
  ground is carried as coverage, not silently dropped
