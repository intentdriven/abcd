---
id: itd-2610301327282266
slug: a-person-reads-abcd-help-or-abcd-help
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd's help fits the screen it is read on

## Press Release

> A person reads abcd --help, or abcd --help --agent, in a laptop terminal or a narrow window and every line fits the screen. Each verb's name stands on its own, its one-line purpose wraps to the terminal's width on a word boundary, and what it writes and what it refuses sit on the lines below, so nothing runs off the edge or breaks mid-word when the terminal wraps it. At 80 columns the whole menu reads top to bottom without scrolling sideways, and a wider terminal uses the width it has.

## Why This Matters

Typed links: `refines`
[itd-146](../shipped/itd-146-abcd-s-help-renders-in-labelled-command.md)
(the labelled command groups this help renders in) and
[itd-2610090831227812](../shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md)
(the person's menu and the `--help --agent` expansion whose lines this
re-lays).

The help is the first thing a person reads, and today it cannot be read on an
ordinary screen. Measured on 2026-10-30 with `COLUMNS=80`: `abcd --help` runs
to 64 lines whose longest is 172 characters, and `abcd --help --agent` to 94
lines whose longest is 218. Every verb is one line holding its name, its
purpose, what it writes, what it refuses and, in the agents block, the page to
read next (`(read commands/<verb>.md)`), so the terminal wraps each line at
its edge, mid-word and under the name column, and the part that says what a
verb writes or refuses lands in an unaligned tail. The width the help is laid
out to is the terminal's, never a fixed one, so a wide terminal is not
squeezed either.

Whether `--help --agent` should write JSON by default was researched on
2026-10-30 and answered no: the agents block stays text, read by a model the
way the host reads a skill listing, with JSON behind `--json`. So this draft
covers the text layout of both listings, which an agent reads too, and the
machine-readable listing is
[iss-2610301320457709](../../../work/issues/open/iss-2610301320457709-abcd-has-no-machine-readable-list-of-its.md).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
