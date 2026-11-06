---
id: itd-2611041410025208
slug: a-person-who-leaves-abcd-s-agents
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611031342011261]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd can say where its runs stalled and when a person stepped in

## Press Release

> A person who leaves abcd's agents working for days can ask afterwards where the runs stalled and how often a person had to step in. abcd answers from a record kept on their own machine, across runs and over a period they name, alongside what the launch ledger says each run cost. The record holds metadata only, never session content, nothing leaves the machine, and each measure exists because a named question needs it.

## Why This Matters

Typed links: `builds_on`
[itd-2611031342011261](../planned/itd-2611031342011261-every-agent-abcd-launches-for-a-session.md)
(the per-session launch ledger, which owns what each agent cost). This draft owns
none of that figure. Token cost per spec and size classes stay with
[itd-2609292107351737](itd-2609292107351737-every-development-session-records-what.md),
and it settles the question the shipped per-session telemetry
([itd-2609091718595846](../shipped/itd-2609091718595846-any-captured-session-can-be-handed-to-an.md))
left open only for stalls and interventions: what a roll-up across sessions is for.

abcd already has the start of this. `implement log` keeps a day's events, among them
interventions, stops, backoffs and agent time, and `implement report` derives
figures from them. It has three gaps for a run left alone for days:

- **Most lines are written by hand.** The report's own coverage check exists because
  hand-kept events stop partway through a run. Only an outage writes an intervention
  by itself, so the runs with the least attention have the thinnest record. Autonomous
  runs have kept separate hand logs of each intervention and hand-rolled step for
  this reason.
- **It reads one day or one log at a time.** Nothing answers "over the last two
  weeks, where did runs stall and what made a person step in?".
- **A build's run state is kept elsewhere.** It lives in the checkout's local tier,
  separate from the shared run log, and a worktree's removal deletes it.

The aim is days or weeks of unattended work, and the honest measure of that is how
often a person was needed and why. Each intervention's reason is also a list of what
abcd should do next.

## What's In Scope

- **What abcd can see, written without a person**: stalls, restarts, refusals and
  hand-backs that already pass through abcd's own verbs.
- **A question over a period**: where runs stalled and when a person stepped in,
  across runs on this machine, beside the ledger's cost figures.
- **A plain mark for what abcd cannot see**: a person acting outside abcd is logged
  by hand as today, and an answer says how complete its record is.
- **A measure per question**: a new measure is added when a named question needs it.

## What's Out of Scope

- Comprehensive collection. Recording everything in case it is useful costs storage
  and privacy, and [DORA warns](https://dora.dev/guides/dora-metrics-four-keys/)
  against measurement at the expense of improvement.
- Anything that leaves the machine, and any background network call, which would
  have to replace [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)
  in the open. Usage sent back across people was discussed and not filed.
- Session content: metadata only, as itd-2609292107351737 already binds.
- Whether products built with abcd get used: not filed here either.
- Scores, targets and per-person figures.
- Tuning configuration from the results, which is itd-64.

## Mechanism

- We expect hand logs of autonomous runs to stop being needed, because most stalls
  and hand-backs already pass through abcd's verbs and can be written where they
  happen. Shown wrong if a person still has to keep a hand log to answer where a run
  stalled, or if most interventions turn out to happen where abcd cannot see them.

## Scope Conditions

- Runs on one person's machine. Several machines or accounts are not combined here.
- Runs driven through abcd's own verbs. A person acting directly in a host or a shell
  is visible only if they log it.
- A few people running abcd, so figures are for a person reading their own runs and
  do not support comparison between people.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **One store or a read across several.** Whether this extends the run log, or reads
  it together with the build's run state and the ledger.
- **What counts as a stall.** No progress for how long, on what signal, and how a
  backoff or a load wait differs from a stall.
- **Survival.** Where a run's record goes so that removing a worktree does not delete
  it.
- **The first named questions.** Which questions earn the first measures, and whether
  "show where work is waiting" (itd-2611041409531723) can read this record too.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
