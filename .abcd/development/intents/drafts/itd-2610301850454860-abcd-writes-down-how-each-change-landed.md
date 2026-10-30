---
id: itd-2610301850454860
slug: abcd-writes-down-how-each-change-landed
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-82, itd-2610150602547271]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd records how each change landed

## Press Release

> abcd writes down how each change landed. When a change reaches the default branch, abcd records with it whether a person approved it, the drain landed it under its rule, or it merged automatically after its checks passed, and which rule let it through. Anyone can then ask which changes landed with no person involved, and a change that is later undone can be traced to the rule that admitted it.

## Why This Matters

Split out of itd-2610301750162657 on 2026-10-30 at the product thinker's
ruling in its planning interview: the spot-check of what abcd decided alone
waits for this record, and the record is useful on its own.

Today abcd cannot tell, after the fact, how a change reached the default
branch. Which issues a drain lane landed lives only in the drain's run state,
an uncommitted file in one worktree's local tier
(`internal/core/implement/loop/drain.go`); every merge commit carries the same
author whether a person, a lane or the merge queue landed it; and whether a
person approved a pull request is known only to the forge's review decision,
which abcd reads at landing (`internal/core/landing/land.go`) and keeps
nowhere. A change later reverted cannot be traced to the rule that let it
through, and nobody can list the changes that landed with no person involved.

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
