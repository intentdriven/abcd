---
id: itd-2609212103572513
slug: an-intent-names-the-release-it-must-land-by-and-the-cut-says
spec_id: spc-2609212138243443
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609211913453478, itd-2609212103568351]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-73]
related_adrs: [adr-2609212115255771]
---

# An intent names the release it must land by, and the cut says whether it did

## Press Release

> **A planned intent may carry `target_release`; the cut reports every targeted intent still unshipped and moves its target forward.**
>
> "Milestones were the only thing that said 'this must be in the next one', and they went with the phases," said a product thinker at a cut. "Now the intent says it, the dry run lists what has not made it, and after the cut the target rolls forward on its own. It never refuses my release; it never lets me forget either."

## Why This Matters

The research pass of 2026-09-21 found the strongest challenge to retiring milestones: a derived release says what shipped, and nothing says what must land before the next cut. Kubernetes answers it with a milestone field on the record rather than a unit; this record is that field, ruled report-and-move-forward, never refuse (adr-2609212115255771, decision 3).

## Mechanism

We expect a target the cut reports and carries forward to answer "what must land" without a planned unit, because the field lives on the record that owns the work and the cut is the moment anyone reads it; shown wrong if targets are never set, or set and carried past two cuts without a word.

## Scope Conditions

None stated.

## What's In Scope

- **The field**: `target_release: vX.Y.Z` (or `next`) on a planned intent, validated as a version, written by `intent plan --target` or `intent target <itd-N> <version>`; the record lint refuses it on a shipped or superseded intent.
- **The report**: `launch --dry-run` and the cut list every targeted intent not yet shipped, in text, in the receipt and in `--json`; the cut proceeds.
- **The move**: at the cut, each unshipped target is rewritten to the next version in the change that rolls the changelog, and the changelog names the move.
- **The board**: the status block marks a targeted intent with its target in Next and Later.

## What's Out of Scope

- A refusing gate.
- A target on an issue.
- Any date.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Report, never refuse (adr-2609212115255771, decision 3).
2. The target moves forward at the cut, so the field never goes stale.
3. The field is optional and lives on the intent alone.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a planned intent, **when** `intent target <itd-N> v0.11.0` runs, **then** the record carries `target_release: v0.11.0`, and the same on a shipped or superseded intent is refused by the verb and by the lint.
- **Given** a targeted intent still planned, **when** `launch --dry-run` or the cut runs, **then** it is listed as targeted and unshipped in text, the receipt and `--json`, and the cut proceeds.
- **Given** the cut is written, **when** the changelog is rolled, **then** each unshipped target is rewritten to the next version in the same change and the changelog names the move.
- **Given** the status block, **when** a targeted intent is listed, **then** its row shows the target.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: milestones are retired today and this is the only place must-land-by survives; we expect the cut's report to be read and the moved target to be acted on; shown wrong if targets are never set or carried past two cuts unremarked
