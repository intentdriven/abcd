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
  hand-back routing per kind; the `related_intents` stamp as the only write; the window
  exit and `--max`; the refusal without the ADR.

## Steps

1. The judgement before a lane opens (scope 3)
   - packages: internal/core/implement, internal/core/capture
   - tests: an eligible issue whose remedy changes a user-visible surface or a trust boundary is handed back before any lane opens; one that changes neither opens a lane
2. The summary's remaining counts (scope 8)
   - packages: internal/core/implement, internal/surface/cli
   - tests: the summary carries every count scope 8 names, in text and --json
3. Merged lanes are finished (iss-2610090936460789)
   - packages: internal/core/implement
   - tests: a lane parked at land whose pushed head is on the default branch is removed and marked done on the next drain move, outside the pacing window
4. One issue or package held back for now (iss-2610090642385898)
   - packages: internal/core/implement, internal/surface/cli
   - tests: an issue on the skip list is passed over with its reason in the plan's passed entries, and the list clears when the drain ends
5. A pull request that leaves the merge queue is put back (iss-2610090642376032)
   - packages: internal/core/implement/loop
   - tests: a landed pull request left unqueued at CLEAN is enqueued, and one dropped after a failed merge-group check is reported with the failed check

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 mixed ledger routed by field | scope 1, 2, 5 |
| 2 judgement only hands back | scope 3 |
| 3 the issue-keyed lane | scope 4 |
| 4 user moment promoted, `related_intents` stamp only | scope 5 |
| 5 trust rule flagged, nothing minted | scope 5 |
| 6 handback stops and routes | scope 5 |
| 7 window exit; `--max` | scope 7 |
| 8 the summary | scope 8 |
| 9 ids by shape | scope 10 |
| 10 the page | scope 10 |
| 11 refuses without the ADR | scope 9 |

## Progress

This section says which pieces of the scope have landed; the spec stays open
until the run itself closes it.

- **Landed (the field-only slice): scope 1, 2, 6 and 9, and the dry-run half
  of scope 8 and 10.** The `remedy:` field (`capture --remedy`, the schema's
  allow-list, the reader and the committed-ledger gate; a record carrying only
  `suggested_fix:` reads that value as its remedy, and no record is rewritten);
  the field-only eligibility rule and the order (`internal/core/capture/eligible.go`);
  `abcd drain --dry-run`, which gives every open issue one disposition with its
  rule and reason, in text and `--json`, and writes nothing; the eligibility
  decision record (adr-2609291342092738) and invariant 19; and the start
  refusal, which names the record when the rule has none and otherwise refuses
  because the issue-keyed lane is not built. The command page is
  `commands/drain.md`, listed in the agents block until the product thinker
  rules on the person's fourteen-verb ceiling.
- **Landed (the remedy required, rulings BX3 and H12 of 2026-09-29):**
  `capture` refuses a new issue without a remedy; the automatic filers (the
  consistency pass, every promoted inbox report) write the one machine
  value `none (filed automatically)`, which the capture surface refuses from a
  person; the dry run lists a record carrying it as ineligible until a person
  writes a real remedy with `capture remedy`, the verb that writes or replaces
  the field on an open issue. A record filed before the rule stays readable and
  is listed as ineligible.
- **Landed (the run): scope 4, 5 and 7, and the run's half of scope 8 and
  10.** `abcd build <iss-N>` and the drain start the implement loop keyed by
  the issue: the key is an issue id by shape, the checks are this rule read as
  the dry run reads it and the peers, the brief is the record with its remedy as
  the work and the reproduce-then-fix definition of done, the validators run
  without the fidelity audit, the receipt must declare the issue fixed, and the
  landing resolves it and opens one pull request (`loop/check.go`,
  `loop/issuebrief.go`, `loop/receipt.go`). A receipt's `handback` (kind,
  reason, home) ends the lane before its validators, its worktree and branch
  discarded and the discarded head recorded. The bare `abcd drain` performs one
  move per invocation (`loop/drain.go`): one lane at a time in the drain order;
  a lane's hand-back routed by kind (a user-visible change promoted with
  `capture promote`, the issue gaining the draft in `related_intents` and
  nothing else; a trust rule flagged with its question, nothing minted; a
  design finding or a second package flagged with its home); every field
  hand-back flagged naming its rule; the drain's window closing into
  `next_eligible_at` in `.abcd/.work.local/run/drain.json`; `--max <n>` ending
  the drain at the cap. The summary is text and `--json`, and a move that opens
  or merges nothing exits 0 saying why.
- **Remaining:** scope 3 (the host judgement over each eligible remedy; the run
  opens a lane for every eligible issue until it lands, and only the lane can
  hand its issue back), and the counts of scope 8 not yet in the summary (the
  spend).
