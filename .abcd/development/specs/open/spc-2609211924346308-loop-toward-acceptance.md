---
id: spc-2609211924346308
slug: loop-toward-acceptance
intent: itd-50
origin: researcher-authored
production_mode: hand-written
---
# loop-toward-acceptance

## Summary

The design record for itd-50 as re-scoped on 2026-09-21: the audit-driven
fix round as the last stage of `abcd build`, bounded by the pace rule,
handing an unachievable intent back to drafts.

## Scope

1. **The stage** in the implement loop's state file: after `intent audit
   ingest`, a not-met criterion sets the lane to `fix-round` with the
   criteria named; the loop renders the fix brief from them and starts a
   fresh implementer through the same runner path as any lane agent
   (criterion 1).
2. **The bound**: the pace configuration's fix-round count (itd-2609201925079472)
   read per lane; the state file counts rounds; exhaustion, or an auditor
   verdict of unmeetable-as-written, sets `unachievable` (criterion 2).
3. **The hand-back**: the intent is moved `planned/ → drafts/` under the
   intent store's lock with `replan_reason: <text>` written and the audit
   notes kept; the spec stays open; the run record and the summary list it
   (criterion 3).
4. **The hand verification**: when every checkable criterion reads met, the
   loop's step interface returns a `verify` step naming the intent; the
   host asks the product thinker one question; the answer is written with
   `intent ready --grounds "<pursued|declined>: <their words>"`; declined
   reopens as in 3 (criterion 4).
5. **Inconclusive**: no fix round, no count, one line in the summary
   (criterion 5).

## Out of scope

- A per-intent `audit_mode`; the loop applies to every lane.
- A verification receipt schema; the grounds entry is the record.

## Approach

Three additions to the implement spec's state machine (`fix-round`,
`unachievable`, `verify`), the fix brief renderer beside the lane brief
renderer, and the hand-back write in `internal/core/intent` (a bucket move
the reclassify verb of itd-34 can also make). The auditor's request gains an
"unmeetable as written" verdict value the ingest validates.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 fix round after not-met, re-audit | scope 1 |
| 2 bounded; unachievable | scope 2 |
| 3 reopened to drafts with the reason | scope 3 |
| 4 hand verification as a grounds entry | scope 4 |
| 5 inconclusive counts for nothing | scope 5 |
