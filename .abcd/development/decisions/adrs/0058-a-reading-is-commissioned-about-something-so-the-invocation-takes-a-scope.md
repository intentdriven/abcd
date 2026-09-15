---
id: adr-58
slug: a-reading-is-commissioned-about-something-so-the-invocation-takes-a-scope
status: superseded
date: 2026-08-31
supersedes: null
superseded_by: adr-2609021016286571
related_intents: [itd-183, itd-184, itd-199]
related_rfcs: []
related_adrs: [adr-55, adr-56]
---

# ADR-58: A reading is commissioned about something, so the invocation takes a scope

## Context

The maintainer rulings of 2026-08-28 closed the cold-reading assembler's
invocation at two operands. Ruling M8 is recorded in itd-183 in these words:

> **No free text at any position — ruled as M8, stricter than proposed:**
> position and target state only. The ids-only comparative argument proposed
> here is superseded; it survives only as the recorded fallback if the
> evaluative position proves to need candidates at invocation.

Brief invariant 15 carries the same closure into the brief: a reading's
invocation "carries no free text (position and target state only)".

The reason for that closure is sound and is not in question here. A free-text
operand is a channel through which the framing of a request could reach a
reading that is supposed to be blind to it. Closing the invocation closed the
channel structurally, which is the shape this repository prefers over a promise.

What the closure did not anticipate is that the object it left the instrument
pointed at is too large to hand to anything. Measured over this repository at
one commit, an assembly is about 9.2 MB and roughly 2.4 million estimated
tokens at every position (`iss-2608311501186646`, and the per-kind figures
itd-198 now reports). Three of the four positions receive a byte-identical item
set (`iss-2608311501240566`), so the interface cannot express what distinguishes
four readings that are about four different things. The instrument is correct
and unusable, and it is unusable because it cannot be pointed at anything.

itd-183's ac-3 anticipated exactly one revision if a need were shown: "a
shape-validated record-id list, never prose", conditioned on the evaluative
position proving to need candidates at invocation, and restricted to the intent
family. itd-199 needs something wider on all three counts. Reading the fallback
as though it already permitted a general scope operand would be a
misreading — the kind this repository has been bitten by before, where a rule's
spirit is cited and its letter quietly exceeded. Hence this ADR rather than a
sentence in an intent.

## Decision

The cold-reading assembler's invocation takes a **third operand, a scope**, in a
closed grammar of three token forms: a record id, a record family, or the name of
a committed preset. M8's "position and target state only" is superseded to that
extent and to no other.

The property M8 and invariant 15 were protecting is restated as the property
that now binds: **no operand carries prose.** That is what the two-operand
closure achieved, and it is what the three-operand interface must continue to
achieve.

Each token form clears that bar, and the third clears it by a different
argument from the first two, which is stated here because conflating them is
how the record would come to overclaim:

- A **record id** and a **record family** are shape-validated closed forms.
  A validator either recognises the token or refuses it; there is no residue
  in which a sentence could travel.
- A **preset name** is not a record-id list and must not be defended as one. It
  resolves into a committed file, and that file may name repository paths — it
  must, because source is 82 per cent of the measured weight and no scope
  expressible in record ids alone can reach it. The safety argument is
  different and, at the boundary this ADR governs, stronger: the expansion is
  reviewed, shape-validated, committed, and inside the dirty gate, so it is a
  repository fact a reader can audit rather than something typed at a prompt.

**No repository path is accepted at the invocation.** A path typed at a prompt
is prose by another name and is refused, which is what keeps the preset file
load-bearing rather than decorative.

## Consequences

- **Brief invariant 15 moves.** Its parenthetical is an exhaustive enumeration
  of the operands, so it is amended to name the scope and to state the
  no-prose property directly. The invariant's other clauses — positive
  inclusion, default exclusion, context isolation, the transcript store's
  enumerated consumer list — are untouched.
- **itd-184's operand pin fires, as designed.** The pinned operand set fails
  closed on any addition so that a new operand must say what it does before it
  ships. This ADR is that statement; the pin is updated in the same change.
- **The preset file joins the dirty gate.** An uncommitted edit to it reshapes
  an assembly as surely as an edit to a record does, and the record-lint
  configuration is already named into that set by exactly this argument.
- **The evaluative position's fallback is not taken.** M8's one anticipated
  revision was conditioned on the comparative position needing candidates at
  invocation. That condition is not what triggered this decision, and itd-199
  makes comparative refuse rather than serve it a corpus that is not its
  object. The fallback therefore remains unexercised and available, and this
  ADR does not consume it.
- **The blindness core is untouched.** A scope changes how much a reader was
  given. It changes nothing about the seven blindness conditions, which all
  four definitions carry verbatim and continue to carry. There is no warm
  definition, and this decision does not create one.

## Status note

**Accepted by the maintainer on 2026-08-31.** It was drafted as `proposed` and
held there deliberately: it supersedes a maintainer ruling and amends a brief
invariant, and neither is an agent's act. The accompanying amendment to
invariant 15 was drafted in the same change and is adopted with it.
