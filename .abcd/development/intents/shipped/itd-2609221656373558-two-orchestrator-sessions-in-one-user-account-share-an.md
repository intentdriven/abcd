---
id: itd-2609221656373558
slug: two-orchestrator-sessions-in-one-user-account-share-an
spec_id: spc-2609221657588816
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091416295622, itd-148, itd-2609201925079472]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609091014076309, itd-2609150819440345, itd-33]
related_adrs: [adr-2609221009491186]
---

# Two sessions share one run, and the run measures which way of dividing the work is best

## Press Release

> **A second orchestrator session joins an autonomous run in the same account without the first waiting on it, work is divided one of three ways a window at a time, and the run's report says which way worked.**
>
> "A third of the pilot's clock was a lane waiting for a slot, and the obvious answer, another pair of hands, was the one thing nobody had measured," said a technical facilitator. "Now a second session joins for a window, takes work by a claim, then by batch, then as the one that reviews and lands, and the log says what each cost: lanes landed, collisions, minutes wasted. The first session never waits for it, and if it dies the run does not notice."

## Why This Matters

The pilot lost 100 of 242 clock minutes to lanes waiting for an agent slot, and the ceiling ruling of 2026-09-22 (two implementers plus one reviewer) addresses the symptom inside one session. Whether a second session helps at all is unmeasured, and the ways of dividing work between two sessions (a claim per record, whole batches, or one session building while the other reviews and lands) have never been compared. The record already holds the pieces: the checkout is the unit of isolation and every change starts in its own worktree, the peer listing lets a session see what its peers hold, and the merge queue serialises what lands. What is missing is the claim, the bounds that keep an experiment from becoming a hazard, and the measurement that makes the answer evidence rather than an impression.

## Mechanism

We expect a second session to add throughput without adding collisions, because the queue already serialises merges and a claim makes the record the unit of exclusion; shown wrong if the measured collisions or the agent minutes wasted on backed-off work exceed the lanes the second session lands.

## Scope Conditions

- Holds for two sessions in one user account on one machine, each in its own worktree from the machine-scoped store; another account or machine is a different question. <!-- cond: cond-2609221657584605 -->
- Holds while the first session is the one that may cut a release; a run with no such session is a run with no release. <!-- cond: cond-2609221657584877 -->
- Holds where the run log is one append-only file per repository per day, each event one line, so two writers do not interleave a line. <!-- cond: cond-2609221657583516 -->

## What's In Scope

- **Joining**: the second session reads the run's state and joins without the first being told; the first never waits on it and never depends on a lane it holds.
- **Three division modes, one per window, in order**: a claim per record; whole batches assigned per session; the first builds while the second reviews, audits and lands. The run log records which mode a window ran, which lanes each session took, and every collision.
- **The claim**: a session claims a record before opening its lane, in a place both sessions read; a claimed record cannot be claimed twice, and a session that stops releases its claims after a stated time so nothing is stranded.
- **The bounds**: only the first session may cut a release or take a lane that recalibrates the reading corpus; the second holds at most ONE lane at a time however work is divided, obeys its own agent ceiling on top of the first's, backs off on any contention with a logged reason, and its stop conditions never stop the first.
- **The measurement**: per mode, lanes landed, wall clock, collisions, agent minutes including those spent on work that backed off, and the ceiling wait each session saw.
- **The report**: the run's summary compares the three modes on those numbers and names which it would keep.

## What's Out of Scope

- A third session, and any session in another account or on another machine.
- A scheduler that assigns work automatically (this run compares modes; a later record may pick one).
- Changing what the queue or the record gates do.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. All three modes are tried and measured; none is chosen in advance (ruled 2026-09-22).
2. One mode per session window, in order, and the second session is optional: the run never depends on it.
3. The second session holds at most one lane, may never cut a release or take a reading-corpus lane, and backs off on contention.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a run in progress, **when** a second session joins, **then** it needs no word from the first, the first never waits on it, and the log records the join.
- **Given** a window, **when** it opens, **then** the log names its division mode, and at the window's end the log carries the lanes each session took and every collision.
- **Given** a record one session has claimed, **when** the other tries to claim it, **then** it cannot, and the attempt is logged; **given** a session that stops holding claims, **when** the stated time passes, **then** its claims lapse and the records are claimable again.
- **Given** the second session, **when** it reaches a release step or a lane that recalibrates the reading corpus, **then** it refuses and leaves it to the first.
- **Given** the second session, **when** it is running, **then** it holds at most one lane, obeys its own ceiling above the first's, and a stop condition it meets stops only itself.
- **Given** contention of any kind, **when** the second session meets it, **then** it backs off and the log names the reason and the minutes spent.
- **Given** a completed run, **when** its report is read, **then** it compares the three modes on lanes landed, wall clock, collisions and agent minutes, and names which it would keep.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-d0980beeafe9 -->
Fidelity review OWED (receipt rcp-d0980beeafe9).

## Grounds

- pursued: the pilot's ceiling wait was a third of its clock and a second session is the cheapest test of whether more hands help before the verb owns pacing; we expect throughput without collisions; shown wrong if collisions and wasted agent minutes exceed the lanes the second session lands
