---
id: spc-2609212138243443
slug: an-intent-names-the-release-it-must-land-by-and-the-cut-says
intent: itd-2609212103572513
origin: researcher-authored
production_mode: hand-written
---
# an-intent-names-the-release-it-must-land-by-and-the-cut-says

## Summary

The design record for itd-2609212103572513: the `target_release` field, its verbs, the cut's report and the move forward.

## Scope

1. **The field and verbs**: `intent target` and `plan --target` in `internal/core/intent`, version-validated through the release package's parser; lint row in `record_schema` (criterion 1).
2. **The report**: `launch.DryRun` and `launch.Ship` read planned intents with a target and list those not in `shipped/` (criterion 2).
3. **The move**: the ship path rewrites each listed target to the derived next version in the receipts commit and appends a changelog line (criterion 3).
4. **The board**: the status block reads the field (criterion 4).

## Out of scope

- Refusal; issues; dates.

## Approach

Small: one frontmatter key, two verbs, one read in launch, one write at the cut; the move is part of the receipts commit so a cut is one change.

## Footprint

- packages: internal/core/intent, internal/core/launch, internal/core/positioning
- tests: the verbs and refusals; the dry-run list; the move at a fake cut; the board row

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 field and refusals | scope 1 |
| 2 report, cut proceeds | scope 2 |
| 3 move forward | scope 3 |
| 4 board | scope 4 |
