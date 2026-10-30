---
id: itd-2610301750162657
slug: a-person-spot-checks-what-abcd-s-agents
spec_id: spc-2610301925429250
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-82, itd-2610150602547271]
related_intents: [itd-2610240850336695]
severity: minor
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2610301850454860]
impact: additive
---

# abcd's unattended work is spot-checked against specs and intents

## Press Release

> abcd's agents build, review and merge work on their own, and no person reads code: a model that did not write a change reviews every change. On top of that, at a window the repository sets (each week, say, or every thousand changes), abcd draws a few items at random, picked so no agent can steer the draw, from what was decided with no person involved in that window: changes that merged, issues the drain fixed or handed back, and the automated facilitator's own decisions. It adds them to a review queue, and nothing autonomous ever waits for that queue. The technical facilitator checks each outcome against its spec and each decision for where it went wrong; where abcd itself is the facilitator, a human facilitator checks a sample of its decisions in turn; the product thinker checks what came back against their intents, each working through the queue when they have time. Each problem found becomes an issue against the rule that let it through, and each spot-check says plainly what a clean result rules out and what it cannot.

_The product thinker's accepted text (2026-10-30, planning interview), replacing the draft's first wording, in which a person read each sampled change._

## Why This Matters

Typed links: `builds_on`
[itd-82](itd-82-drain-ledger-triage.md) (the drain, whose taken
work is sampled and whose written rule,
[adr-2609291342092738](../../decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when.md),
is what a finding about it amends) and
[itd-2610150602547271](../shipped/itd-2610150602547271-a-repository-declares-whether-its.md)
(the merge mode under which a pull request that passed its gates merges with
no person asked: "the reviews are automated, not skipped"). Related:
[itd-2610240850336695](../drafts/itd-2610240850336695-before-stepping-away-the-person-grants.md),
whose grant lists each act a session took under it; that list reports every
act, while this one reads a drawn few of them.

abcd lets agents change a repository with no person reading the change: the
drain fixes what its rule says needs no decision, and a pull request that
passes its gates can merge unattended. Nothing then looks back at what went
through. In the 30 days to 2026-10-30, 488 changes landed on abcd's own
default branch (merge commits on its first-parent line), and the product
thinker ruled on 2026-10-16 that its drain rule be widened to every severity
([iss-2610161655393137](../../../work/issues/open/iss-2610161655393137-abcd-s-own-drain-rule.md),
open, not yet applied), with no check planned of what the wider rule lets in.

The state-of-the-art pass of 2026-10-30 is the evidence, in
[`2026-10-30-spot-check-unattended-changes-sota.md`](../../research/notes/2026-10-30-spot-check-unattended-changes-sota.md).
The mechanism below rests on four of its findings: a sample is only fair when
every item has a chance of selection and the system being checked does not
choose it (ISA 530; Lakkaraju et al., KDD 2017); zero findings in n reads
rules out only a failure rate above about 3/n (the rule of three); a reader
who sees the automated verdict first leans on it (Parasuraman and Manzey,
2010; Singh et al., FSE 2026); and unattended agent changes do get reverted
after merging (2.66% of agentic pull requests, Oukay et al., MSR 2026).

Background, not evidence: Managers report the same strain from their side.
An HBR article (Fosslien and West Duffy, "Managers Are Struggling to Keep Up
with the AI Productivity Boom", 25 May 2026, reprint H096N7,
https://store.hbr.org/product/managers-are-struggling-to-keep-up-with-the-ai-productivity-boom/H096N7)
describes an "always-on review environment" and advises using AI to pick which
work merits careful review. A Nielsen survey of 1,700 managers for
Monday.com, reported by TechCrunch on 21 October 2026
(https://techcrunch.com/2026/10/21/ai-promises-productivity-but-for-managers-its-mostly-a-headache-for-now-says-new-survey/),
found that nearly half spend more time "reviewing and guiding" their teams'
output, and 46% worry about losing control of agents' actions.

## Mechanism

- We expect abcd's rules for deciding alone to improve over time, because a
  draw no agent can steer gives every unattended decision a chance of being
  checked, and each finding is filed against the rule instead of written off.
  Shown wrong if reverts or corrections of unattended work keep tracing back
  to rules that no spot-check finding named.

_The product thinker's accepted claim (2026-10-30, planning interview)._

## Scope Conditions

- A repository where agents land work and make decisions with no person <!-- cond: cond-2610301925420166 -->
  involved: drain work, automatic merges, or abcd as the technical
  facilitator.
- abcd records how each change landed (itd-2610301850454860). <!-- cond: cond-2610301925420665 -->
- People judge outcomes against specs and intents, never code. <!-- cond: cond-2610301925424008 -->
- More unattended work than anyone could check in full. <!-- cond: cond-2610301925423967 -->

## Acceptance Criteria

Each confirmed by the product thinker on 2026-10-30, one at a time.

- **Given** the work abcd decided alone in a spot-check window, **when**
  abcd draws, **then** the draw is seeded from a public random value fetched
  when the spot-check starts and stored with it, every item in each group has
  a chance of being picked, and the same stored value and items always give
  the same list.
- **Given** the groups (drain fixes that landed, drain hand-backs, automatic
  merges, and the automated facilitator's own decisions), **when** the draw is
  made, **then** it picks up to the repository's set number per group, ten by
  default; half that after three clean spot-checks in a row; the full number
  again after any finding.
- **Given** a drawn item, **when** it is put to a person, **then** it is never
  code: the technical facilitator sees the outcome beside its spec, or the
  decision beside its rule; a human facilitator sees a sample of the automated
  facilitator's decisions; the product thinker sees what came back beside the
  intent; and any automated verdict is shown only after the person has judged
  the item.
- **Given** an item judged wrong, **when** the person says why, **then** abcd
  files an issue naming the item, its group and the rule that let it through,
  and abcd itself never edits that rule.
- **Given** a spot-check with no findings, **when** it closes, **then** abcd
  states per group what the clean result rules out (with ten items, a failure
  rate above about one in four) and never reports it as no problems.
- **Given** a repository's spot-check window (a span of time or a number of
  landed changes), **when** a change lands and the window has passed,
  **then** abcd draws from the work decided alone in that window and adds the
  items to the review queue, and when the window has not passed it draws
  nothing.
- **Given** items waiting in the review queue, **when** an autonomous run,
  a landing or a drain proceeds, **then** none of them waits for the queue,
  however long it is.
- **Given** a review queue holding draws from several windows, **when** a
  technical facilitator sits down to it after weeks or months, **then** every
  item drawn since is still there, grouped by window and group with the
  oldest first, and they can stop at any item and resume later where they
  left off.
- **Given** an item reviewed long after its window, **when** it is shown,
  **then** it gives the item's state when it was drawn and its state now (for
  example since reverted or changed again), and a finding on it is filed
  against its rule like any other.
- **Given** windows whose items are not yet all reviewed, **when** the next
  window's sample size is set, **then** only fully reviewed windows count
  towards the clean run that halves it, and an unreviewed window counts as
  neither clean nor a finding.
- **Given** items waiting in the review queue, **when** the status board
  renders, **then** it says how many wait, for whom, and how old the oldest
  is.
- **Given** a drain run, **when** its summary is shown, **then** it states the
  hand-backs per pull request it opened.
- **Given** any change an agent writes, **when** it is reviewed, **then** a
  model other than the one that wrote it reviews it, and the record names
  both.

## Decisions

Ruled by the product thinker on 2026-10-30, in the planning interview (the
draft text above predates these rulings and is revised to them before it is
planned):

1. **The spot-check also reads the hand-backs**, judged as decisions (was the
   drain right to hand this back?), beside the changes that landed: what the
   drain fixed on its own and what merged with no person asked. Chosen over
   reading only the changes that landed.
2. **Ten items per group by default, configurable per repository.** Chosen
   over about thirty per group; ten clean reads rule out only a failure rate
   above about one in four.
3. **Two lanes: collection and review.** The product thinker ruled that the
   spot-check runs as two lanes that never hold each other up. The collection
   lane draws the sample at a window the repository sets, a span of time (each
   week, say) or a number of landed changes (every thousand, say), and adds the
   drawn items to a review queue. It needs no scheduler: whenever a change
   lands, abcd checks whether the window has passed and, if it has, draws from
   that window. The review lane is the people working through the queue
   whenever they have time; items wait there as long as they need to, and no
   autonomous run, landing or drain ever waits for the queue. This replaces
   two earlier choices of the same day: waiting for a timetable feature, and
   a spot-check a person starts by hand.
4. **The number of reads adapts to recent results**: half as many after three
   clean spot-checks in a row, the full number again after any finding.
5. **No person reads code.** The product thinker ruled that no person in
   abcd's roles has to read code, since sparing people that is the point of
   abcd, and that neither the product thinker nor the technical facilitator
   could judge it. The reading is therefore layered. A model that did not
   write a change reviews its code, and may as well review all of it rather
   than a sample. The technical facilitator, whether a person or abcd,
   spot-checks how outcomes match their specs and where autonomous decisions
   went wrong. A human technical facilitator spot-checks the automated
   facilitator's decisions, to keep its judgement sound. The product thinker
   takes part only in matching what came back against the intents, which
   are their definition of good.

6. **The draw is seeded from a public randomness beacon** fetched when the
   spot-check starts (for example drand), stored with it so the list can be
   recomputed. Chosen over a phrase a person types, which a host's scheduled
   routine has no person to type, and which a person could retype until they
   liked the list. Fetching the value is the documented meaning of the
   spot-check, so abcd still asks the network nothing on its own (brief
   invariant 7).
7. **How each change landed is its own feature**, itd-2610301850454860, which
   this intent waits for: today which work the drain landed lives only in an
   uncommitted local file, every merge commit has the same author, and whether
   a person approved is known only to the forge.
8. **The spot-check earns its place** under the less-but-better principle: a
   few drawn checks replace checking every autonomous decision, or holding
   back from giving the agents more freedom.
9. **The verification ladder (itd-173) stays a separate feature, planned
   apart**: it sets how much checking a change gets before it merges, while
   the spot-check looks back afterwards.
10. **This intent is not bundled with the autonomy grant**
    (itd-2610240850336695). The product thinker first bundled the two, since
    the grant's report lists every act a session took and the spot-check
    draws from the same acts, then took the spot-check out the same evening,
    because the grant's bundle cannot be planned yet (its other members have
    open questions and unapplied review findings) and the spot-check is
    wanted ready to build. It draws from abcd's record of how each change
    landed (itd-2610301850454860), not from the grant's report, and may ship
    before it.
11. **The count of hand-backs per pull request is part of this intent**, as
    the measure of the load the agents put on people; counting hours spent
    reviewing was ruled out, since abcd cannot measure it without watching
    people or asking them to report.
12. **No dependency on scheduling** (decision 3): A landing is the trigger,
    so the spot-check does not wait for itd-13; the record of how each change
    landed (itd-2610301850454860) is its one prerequisite.
13. **Every change is reviewed by a model other than the one that wrote it**
    (the last criterion). abcd already records each reviewer's model class
    (itd-2610240922461281) but does not guarantee it differs from the
    writer's.

15. **Review may come rarely and in bulk.** The product thinker ruled that
    the review lane may be one sitting every other month in which the
    technical facilitator works through a hundred or more items covering
    many windows, and that such a review is still worth having, since its
    findings still improve abcd's rules. So no item expires or is dropped,
    the queue is ordered by window and group with the oldest first, a sitting
    can stop and resume at any item, a late item shows its state then and now,
    and the sample-size adaptation of decision 4 counts only fully reviewed
    windows.

Settled by the record, so no question was put (2026-10-30):

14. **Impact: additive.** A new capability that changes nothing existing
    users rely on.

## Open Questions

_All resolved at the planning interview of 2026-10-30, in `## Decisions`
(the reading list's size and cadence, who reads it, and the hand-back count)._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: abcd is moving to agents that decide and merge alone with no person reading code; we expect a small spot-check no agent can steer to catch the rules that let bad work through before those rules are trusted with more, and we are wrong if spot-checks keep coming back clean while unattended work is still reverted or corrected.
