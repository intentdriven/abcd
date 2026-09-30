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

## Progress

The spec stays open until every criterion has landed.

- **Landed: criterion 1.** The validate stage of `abcd build`
  (`internal/core/implement/loop/validate.go`, spc-2609202134338445 piece 8)
  hands a round that did not pass to a fresh implementer briefed on the
  findings, and the next round re-runs every validator, the audit included.
  Under ruling DQ1a (decision 5) an undecided criterion fails the round as a
  not-met one does, and the fix brief names it as undecided.
- **Landed in part: criterion 2.** The bound: the run's fix-round cap
  (`--fix-rounds`, `pace.fix_rounds`, bundled 3; decision 6) is resolved with
  the pace and kept in the run's state, and a round that does not pass once
  the lane has taken the cap stops the lane at the `handed-back` stage with the
  verdict unachievable and the last round's findings; the run starts nothing
  further for it (`internal/core/implement/loop/handback.go`). Not built: the
  auditor's "unmeetable as written" verdict value.
- **Not built: criterion 3.** The hand-back leaves the intent in `planned/`:
  the move to `drafts/` with `replan_reason` and the run summary's replan list
  wait on the landing (spc-2609202134338445 piece 9).
- **Not built: criterion 4**, the hand verification.
- **Landed in part: criterion 5.** An audit return the loop cannot read as a
  verdict is refused, records nothing, starts no fix round and counts against
  nothing; the run summary that names it waits on the run record's rendering
  (spc-2609202134338445 piece 10).
