---
id: spc-2609212015054359
slug: drain-ledger-triage
intent: itd-82
origin: researcher-authored
production_mode: hand-written
---
# drain-ledger-triage

## Summary

The design record for itd-82 as revived on 2026-09-21 (decisions 1 to 8)
after two adversarial reviews: `abcd drain` classifies the open ledger by
field, hands the eligible issues to the implement loop's issue key one at a
time, routes the rest back by kind, and runs until nothing eligible is left
under the pace rule.

## Scope

1. **The `remedy:` field**: `capture` gains `--remedy "<text>"` and the
   frontmatter key (the optional `suggested_fix` made first-class under this
   name; a migration reads the old key as the new one where present); the
   issue schema names it; the surface page documents it (criterion 1).
2. **Eligibility, by field** (`internal/core/capture/eligible.go`): open,
   nothing unshipped in `blocked_by`, `remedy:` present, category in the
   fixable set (`bug`, `documentation`, `drift`, `inconsistency`,
   `tech-debt`, `ux`), severity `minor` or `nitpick`; every other open issue
   receives its disposition by the rule that excluded it (criterion 1).
3. **The judgement**: a host pass over each eligible issue's remedy asking
   one question, "does this remedy change what a user sees or a trust
   boundary", returning a validated yes/no with a reason; yes hands back,
   no changes nothing (criterion 2).
4. **The lane**: `implement.Start` with key `iss-N` (decision 10 on the
   parent), one at a time under the ceiling; the loop's own definition of
   done, resolve and pull request apply (criterion 3).
5. **The hand-back, by kind**: user moment → `capture promote <iss-N>`
   (itd-119's verb) and nothing else written; trust rule → a flag in the
   summary with the question; above severity or outside the fixable set →
   a flag naming the rule; other → a flag with the proposed home; a
   `handback:` on a lane report → the lane discarded and the same routing
   (criteria 4, 5, 6).
6. **Order**: `tech-debt`, `documentation`, `inconsistency`, `drift`,
   `bug`, `ux`; `nitpick` before `minor`; oldest first (decision 7).
7. **The clock and the caps**: the pace rule's window and ceiling; at the
   window's end the state file takes `next_eligible_at` and the process
   exits; `--max <n>`; the default is all (criterion 7).
8. **The summary**: every disposition with its reason, the order rule,
   every cap, in text and `--json`; a run without a merge exits 0
   (criterion 8).
9. **The decision record**: the spec's first delivery mints the ADR for the
   eligibility rule with `abcd decide` from decision 4 and scope 2, adds
   the brief invariant, and the run refuses to start unattended without
   them (criterion 11).
10. **Safety and pages**: issue ids validated by shape; the command page
    (criteria 9, 10).

## Out of scope

- The lane's internals (the parent); picking among intents (the sibling).
- Writing a classification onto the issue (decision 8).

## Approach

`internal/core/capture` gains the field and the eligibility function;
`internal/core/implement` gains `Drain`, a loop over the ordered eligible
set that calls `Start` with the issue key and reads each lane's outcome; the
judgement follows the host-pass shape (request file, validated return); the
hand-back writes go through `capture promote` and the summary writer. The
ADR is minted in the first delivery and reviewed with the diff.

## Footprint

- packages: internal/core/capture, internal/core/implement, internal/core/issueschema, internal/surface/cli
- tests: the eligibility table over a fixture ledger; the order; the
  hand-back routing per kind; `promoted_to` as the only write; the window
  exit and `--max`; the refusal without the ADR.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 mixed ledger routed by field | scope 1, 2, 5 |
| 2 judgement only hands back | scope 3 |
| 3 the issue-keyed lane | scope 4 |
| 4 user moment promoted, `promoted_to` only | scope 5 |
| 5 trust rule flagged, nothing minted | scope 5 |
| 6 handback stops and routes | scope 5 |
| 7 window exit; `--max` | scope 7 |
| 8 the summary | scope 8 |
| 9 ids by shape | scope 10 |
| 10 the page | scope 10 |
| 11 refuses without the ADR | scope 9 |
