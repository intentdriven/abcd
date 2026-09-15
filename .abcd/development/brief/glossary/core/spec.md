---
term: spec
bounded_context: core
definition: A specced block of work in abcd's native spec store that implements one or more intents, broken into ordered tasks with acceptance criteria.
aliases: ["implementation spec"]
forbidden_synonyms: ["sprint", "milestone", "project", "feature", "epic"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/intent
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# spec

A **spec** is the implementation unit in abcd's workflow. Where an intent
describes *what* and *why* in human/customer terms, a spec describes *how* in
technical terms. Each spec lives in the **native spec store** as a directory
whose location encodes its status, decomposed into numbered tasks; the companion
harness `ccpm` is the primary deeper backend
([adr-26](../../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)).

## When to use

Use "spec" when referring to a specced technical work block with an `spc-N`
identifier. Specs have acceptance criteria, task lists, and are driven to
completion by the pluggable run seam
([adr-27](../../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)) or direct
implementation.

## When NOT to use

Do not call a spec a "feature" (too generic), "sprint" (carries Scrum cycle
connotations), or "milestone" (a milestone is the end condition of a
[phase](phase.md), not an individual work block) or "phase" (a phase bundles many
specs).

## Examples

- "Spec `spc-3` implements the grill skill and glossary infrastructure."
- "Task `spc-3.1` is the first implementation task of that spec."

## Related terms

- [intent](intent.md) — the human-authored document that a spec realises
- [phase](phase.md) — the arc that bundles many specs

> **Open question (adr-35):** this entry previously related a spec to a *voyage*, glossed as "a full
> lifecycle that contains many specs". [adr-35](../../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)
> retires that sense: [`voyage`](voyage.md) is now the operations namespace at
> `~/.abcd/voyage/<source-root-sha>/`, and it contains no specs. Whether abcd still needs a term for
> the end-to-end project lifecycle — the container the old gloss reached for — is not settled by
> adr-35 and is not decided here.
