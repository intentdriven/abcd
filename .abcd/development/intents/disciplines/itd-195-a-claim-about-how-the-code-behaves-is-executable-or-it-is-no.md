---
id: itd-195
slug: a-claim-about-how-the-code-behaves-is-executable-or-it-is-no
spec_id: null
kind: discipline
kind_notes: "A rule every change inherits: prose that states how the code behaves is either backed by something that runs, or it is not written."
suggested_kind: discipline
reclassification_history: []
builds_on: []
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A claim about how the code behaves is executable, or it is not made

## The rule

A comment, doc or record that states **how some part of the codebase behaves**
is backed by something that runs, or it is not written.

The forbidden shape is a list: "these are the readers and this is what each
does", "these are the places this value is copied", "every surface handles this
the same way". Such a list is words. Nothing exercises it. When the code moves,
the list stays and becomes false, and it is believed *because* it is specific.

Where the fact is worth stating, state it as a test that establishes it by
running the thing. Where it is not worth a test, do not state it: say what the
code in hand does and stay silent about the rest.

## Why

Measured, not reasoned. On 2026-08-30 two unrelated claim families on two
unrelated branches each took FOUR rounds to correct, and each correction
introduced a fresh error:

- an enumeration of what nine record stores' readers do with a duplicated key;
- an enumeration of where a three-token vocabulary is copied.

The error rate did not fall with effort, which is the signature of a format
problem rather than carelessness. A correction to an unverifiable list is itself
unverifiable, so each round produced a new claim nothing could check.

Making the first one executable settled it in one round AND explained the four
failures. The written version had one row per store; several stores have
several readers, and those readers DISAGREE — one refuses a malformed record,
another quietly keeps the first value. A list shaped one-row-per-store cannot
hold that, so every hand-written version had to drop a row. Four rounds of
checking whether rows were right never asked whether the shape could carry the
truth. The executable version has fourteen rows for nine stores.

It then caught two false statements in the orchestrator's own briefing, which
four review passes reading prose had not.

## The gate

- **Given** a comment, doc or record stating how a component behaves, **when**
  it is reviewed, **then** it either cites something that runs and would fail if
  the statement stopped holding, or the statement is removed rather than
  corrected.
- **Given** such a claim found to be wrong, **when** it is fixed a second time,
  **then** the fix is to make it executable or delete it, never to reword it
  again.
- **Given** an enumeration made executable, **when** a new member appears,
  **then** a gate fails rather than the list ageing.

## Fit

This is the general rule; two adopted principles are its special cases, and
neither covers it.
[enforcement-claims-are-facts](../../principles/enforcement-claims-are-facts.md)
forbids describing a gate that does not run — a claim about EXISTENCE, judged
when written. This rule is about a claim STAYING true: a comment can be exactly
right the day it is written and false six months later, which that principle
does not reach because it was a fact at the time.
[guards-prove-themselves](../../principles/guards-prove-themselves.md) requires a
refusal path to ship with a test that watches it refuse — the same instinct,
scoped to guards. Filing the general form beside two adopted specialisations is
structure rather than a third copy, and if it is adopted the two should say they
are instances of it.

## Staging

**Adopted 2026-08-31 as a STUB.** The rule stands and the two adjacent
principles now point at it as instances. Nothing else is worked through: the
sweep is unstarted, the gate is prose rather than a detector, and the
relationship between this record and its two special cases is asserted rather
than reconciled. Deliberate — the cold-reading experiment has the floor, and
this waits for it.

The rule binds new work immediately. The existing corpus does not conform and is
not assumed to: a sweep of the whole codebase for prose that states how other
components behave is the work this intent carries, and it has not been done. Two
sites are already converted (the record-store reader table) or removed (the
vocabulary copies); everything else is unsurveyed.

The sweep is deliberately not automated first. What counts as "a claim about
behaviour" versus ordinary description is a judgement the corpus has to teach,
which is [script-first-mvp](../../principles/script-first-mvp.md): survey by hand,
grade what is found, and let the detector follow the corpus rather than lead it.
