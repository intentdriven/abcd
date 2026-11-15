---
id: itd-2611041409531723
slug: when-a-product-thinker-wonders-why
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610301850454860]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Asked where work is waiting, abcd shows the wait at each stage

## Press Release

> When a product thinker wonders why promised work is taking so long, they can ask abcd where it has been waiting. abcd reads the history of its own records and shows, for each stage committed work passes through, how long planned intents and open issues have waited, what is still open, and the spread rather than one average. Review and merge-queue time come from the landing record, or from the forge when the person asks for that read. abcd names the largest measured wait as the first place to look, never as the proven constraint, and it keeps no score, target or per-person figure.

## Why This Matters

Typed links: `builds_on`
[itd-2610301850454860](../shipped/itd-2610301850454860-abcd-writes-down-how-each-change-landed.md)
(how each change landed, the source for review and queue time that no record
holds today). The one constraint this repository has written down sits inside a
stage, [iss-2610161639025702](../../../work/issues/open/iss-2610161639025702-preflight-throughput-sets-the-pace-of.md),
which is why folder ages alone cannot name a constraint.

Faster agents move the wait instead of removing it, and nothing in abcd says
where it has moved to. The status board counts what is ready and what is parked,
and shows no ages. A hand count on 2026-11-04, from the default branch's
first-parent history, put a median of 8.2 days between an intent being planned
and shipped (115 intents), with 33 planned intents still open at a median age of
18 days. Over half of intent entries landed in batches, so these figures are good
to about a day at best. A question that took a script and a review to answer
should take one ask.

DORA's [value stream guide](https://dora.dev/guides/value-stream-management/)
tells teams to find the top constraints on flow, and that saving minutes does not
matter in a process that takes days. Its
[metrics guide](https://dora.dev/guides/dora-metrics-four-keys/) warns against
metrics as goals, single headline numbers and measurement at the expense of
improvement, which is why this intent reports waits and sets no target.

## What's In Scope

- **Stage waits on an explicit ask**: derived from when a record entered each
  folder, counting folder entries and not later edits, for planned intents and
  open issues. What is still open is shown beside what has finished, so neither
  skews the other.
- **The spread, with its sample size**: never one average alone.
- **Review and queue time**: from the landing record, or from the forge only when
  the person asks for that read (adr-38).
- **A first place to look**: the largest measured wait, named as a lead and never
  as the proven constraint.

## What's Out of Scope

- Drafts as a queue. A draft is a bench of ideas, and counting it would make drafts
  the largest wait by construction. The count of drafts stays on the board.
- Any score, target, trend goal or per-person figure, and merge counts as a
  headline number: agents can inflate them at no cost.
- A board line that names the constraint automatically from folder ages.
- Five stages from git history alone. Approvals and queue entry are in no record,
  so that first wording was shown false.

## Mechanism

- We expect the first place the person looks to change, because the largest wait
  is often not where attention already sits and today only a hand count shows it.
  Shown wrong if, over several asks, the named wait only ever repeats what the
  person already believed, or if looking there changes nothing they do.

## Scope Conditions

- A repository with enough planned intents and resolved issues to show a spread;
  below that, abcd says the sample is too small and gives the counts.
- Timings are as coarse as landings are: a batched landing makes several records
  appear to move together.
- One repository at a time. Waits are not compared across teams or repositories,
  which DORA warns against.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **Where it is asked.** An explicit verb of its own, or an extension of the
  planned landings listing, with at most a disk-only line on the board.
- **Issue stages.** Issues have only open, resolved and wontfix, so what counts as
  an issue's stages is unsettled.
- **Build start.** No record marks it, and the first branch commit is lost on a
  squash merge.
- **Gate wait against gate run time.** Neither is in the records; whether this
  intent or the landing record should split them is open.
- **Backlog against blocking queue.** A rule for telling work nobody has chosen to
  start from work that is held up.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
