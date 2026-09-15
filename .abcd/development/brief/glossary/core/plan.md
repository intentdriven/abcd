---
term: plan
bounded_context: core
definition: The maintainer's sign-off act `abcd intent plan <itd-N>`, which mints a spec, links both sides and moves a draft intent to planned/. Three further senses share the word — the ordered build plan the phase docs hold, a dated design plan under development/plans/, and a session's planning brief — and each is qualified where it appears.
aliases: ["intent plan", "planning sign-off"]
forbidden_synonyms: ["approve", "estimate", "schedule", "backlog"]
status: stable
introduced_in: itd-80
starts_when: null
ends_when: null
not_to_be_confused_with: core/spec
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# plan

Unqualified, **plan** is the verb: `abcd intent plan <itd-N>`. The invocation *is* the
maintainer's sign-off on an intent's acceptance criteria — never run unattended, never inferred
from consent. It mints the spec stub, links intent and spec, stamps an identity onto every
unmarked scope condition, and moves the record `drafts/ → planned/`. The surface page is
[`commands/intent.md`](../../../../../commands/intent.md); the rule it serves is that no
implementation begins without a planned, specced intent.

## Senses

Four senses share the word. The glossary fixes the bare form for the verb and requires a
qualifier on the other three (iss-2609012245352480).

| Sense | The one spelling | Where it lives |
|---|---|---|
| The sign-off act that mints a spec and moves an intent to `planned/` | **plan**, bare, or `abcd intent plan` | [`commands/intent.md`](../../../../../commands/intent.md), [`04-surfaces/05-intent.md`](../../04-surfaces/05-intent.md) |
| The ordered sequence of phases the project builds in | **the build plan** | [`roadmap/README.md`](../../../roadmap/README.md), which calls the phase docs "the ordered build plan" |
| One dated design or implementation document, `YYYY-MM-DD-*.md` | **a design plan** (the `plans/` family) | [`development/plans/`](../../../plans) |
| The per-intent summary a host session writes before the planning interview | **a planning brief** | `.abcd/.work.local/scratch/planning-briefs/`, described in [`commands/intent.md`](../../../../../commands/intent.md) |

**The state and the act carry the same word, deliberately.** An intent in `planned/` is one the
verb has moved there, so "planned" as a lifecycle bucket and "plan" as the act are one sense,
not two.

**Where the record disagrees.** The draft intent
[itd-41](../../../intents/drafts/itd-41-phase-negotiator.md) uses "plan" for a *proposed*
ordering — "presents it as a proposal to edit or reject — not as a committed plan" — which is
the build-plan sense used of something not yet committed to. The glossary reads that as the
build-plan sense, qualified; it is not a fifth meaning.

## When to use

Use bare "plan" only for the intent verb and its sign-off. Qualify every other use: "the build
plan", "a design plan", "a planning brief".

## When NOT to use

Do not use "plan" for a spec — the spec is what `plan` mints, and it is the design record the
build works against. Do not use it for a phase's `## Scope`, which records bundling rather than
sequence, and do not use it as a synonym for a schedule or an estimate: abcd's records carry no
dates for future work.

## Examples

- "`abcd intent ready itd-N` exits 1, so the intent cannot be implemented; offer the planning interview and let the human run `plan`."
- "The phase docs are the ordered build plan; the roadmap dashboard reads live state instead."
- "The design plan for the Go rebuild is a dated document under `development/plans/`."

## Related terms

- [intent](intent.md) — what `plan` acts on
- [spec](spec.md) — what `plan` mints
- [phase](phase.md) — the sequencing unit the build plan orders
- [roadmap](roadmap.md) — where the build plan and the dashboard both live
