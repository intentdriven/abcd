---
id: itd-2611241456272313
slug: abcd-decides-for-each-change-which
spec_id: spc-2611241502223513
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611011325421304]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2611011325421304]
---

# Each change gets one check budget, and the check on what lands is never cut

## Press Release

> **abcd decides, for each change, which checks it needs and where each one runs, so a change is checked in full once, on exactly what will land, and only as much earlier as it helps.** A project often checks the same change three times: on the author's machine, again when the change is proposed, and a third time on the merged result. abcd keeps the last of these whole for every change, because it is the only one that tests what actually lands. The earlier ones become early warnings, sized to the change: a change to documentation gets the quick checks, a change to one part of the code gets that part's tests, and nothing the author's machine has just run is run again an hour later for the same answer. Before a change is pushed, abcd says in plain words which checks will run where, which are left to the final check, and which platforms no check covers until a scheduled run. It also says what it is saving first: waiting time in a public project, paid checking time in a private one, unless the project says otherwise. Where a project has only one check, abcd changes nothing and says so.
>
> "Every change I made was tested three times, and the third time passed 99 times in 100," said Maya, who maintains a small project with two friends. "Now each change is tested properly once, I get the quick answer in minutes, and when a change does need more, abcd tells me why."

## Why This Matters

A person whose project abcd manages is not a CI engineer. When checks are slow, the usual answers are more machines or fewer checks. Both miss the cause, which is repetition: the same tests run on the same change at several points, because each point was set up on its own and none knows what the others ran.

abcd's own repository shows the cost. On 2026-11-24 a change there was tested three times: about 50 minutes on the author's machine, about an hour when proposed, and 40 to 60 minutes on the merged result. The third run passed 195 times in 200, so the first two had already told the author almost everything. The repetition also multiplies bad luck: a test that fails now and then has three chances to hold up each change, and on that day three such tests cost a release about seven hours.

Parts of the answer are recorded already: reuse of unchanged results on the person's machine (itd-2611011325421304), the layout of hosted checks around a merge queue (itd-2610110940216682), landing where the forge tests no merge (itd-2610261025516272), and a machine the person owns as an independent checker (itd-2611011325430933). Each cuts one cost at one point. None says which point a given check belongs to for a given change, so two of them can each remove the same safety, or each keep the same waste. In abcd's own repository that decision exists twice today, written by hand in two places with two different lists of what counts as a light change. A project abcd manages has neither.

## Mechanism

We expect that once one declaration says which checks a project has, what each reads and where each may run, abcd can plan each change so that any check may move later and none is ever lost, and that the plan it states before the push is what then runs, because the check of the merged result runs every declared check for every change and is the only verdict a landing rests on (adr-2611241456284353): a wrong or cautious plan costs a later warning or a repeated run, never an unchecked landing. Shown wrong if, over fifty landed changes in a project that declares its checks, (a) any change reaches the default branch without every declared check having run on the tree that landed; or (b) the plan abcd stated before the push differs from the runs the change then had for more than 1 change in 20; or (c) the person's machine and the hosted run class the same change differently; or (d) in a project with only one check per change, anything about its checking changed; or (e) where an early run has been cut, the final check fails more often than the project's limit and the next plan does not restore the early run.

The saving in time and paid minutes is not claimed here. It is claimed and measured where a run is actually cut: the proposal run in itd-2610110940216682, the hosted run in itd-2610261025516272.

## Scope Conditions

- Holds for a project in which a change is checked at two or more points before it lands. With one point, or none, the budget changes nothing and says which. <!-- cond: cond-2611241502222226 -->
- Holds where one of those points tests the merged result before the default branch moves and a change cannot land without it: a merge queue, a required check with the branch kept up to date, or the landing check of itd-2610261025516272. Where no point tests the merged result, no run is cut, and abcd says that nothing tests what lands. <!-- cond: cond-2611241502222361 -->
- Holds for any forge and any toolchain. The budget reads which checks exist, the files each reads and the commands that run them from one declaration in the project; it does not know a language. A check with no declared inputs is treated as reading everything. <!-- cond: cond-2611241502226961 -->
- Holds for changes by people with write access and the agents working for them. A change from anyone else gets every check at every point. <!-- cond: cond-2611241502224585 -->
- Does not hold for a release: a release runs every check, on every platform the product ships to, with nothing reused and nothing cut. <!-- cond: cond-2611241502228874 -->
- A platform with no check per change (one checked only on a schedule, for example) is stated as such in every plan, never implied as covered. <!-- cond: cond-2611241502224295 -->

## Acceptance Criteria

- **Given** a project whose checks and the files each reads are declared, **when** the person or an agent asks what a change needs, **then** abcd lists each check, the point at which it will run for this change, each check left to the final check with the reason, each platform no check covers before landing, and whether it is saving waiting time or spend and why.
- **Given** any change at all, **when** it reaches the final check on the merged result, **then** every declared check runs there in full, whatever the change touched and whatever ran earlier, and a hosted definition that would skip one on that run is reported as a defect before it lands.
- **Given** a change for which abcd cannot prove which checks it could affect (an unreadable diff, an undeclared check, a missing base, a classifier that stopped early), **when** the plan is made, **then** the plan is every check at every point, and the reason is shown.
- **Given** a change that edits the declaration of checks itself, **when** its plan is made, **then** the plan is read from the default branch's declaration, not the change's, and every check runs at every point.
- **Given** a change from someone without write access, **when** it is proposed, **then** every check runs at every point, and nothing its author reports shortens any of them.
- **Given** one change, **when** the person's machine and the hosted run each decide how light it is, **then** both read the one declaration and reach the same answer, and a hand-kept list that differs from the declaration is refused before it lands.
- **Given** a project that has set no priority, **when** a plan is made, **then** abcd saves waiting time first in a public project and spend first in a private one, and the plan says which it picked and why; **given** the project sets a priority, **then** that setting is used and named. Under lowest spend no run on a platform the forge prices above the project's cheapest is placed on every change unless the person chose it.
- **Given** a project whose early run has been cut, **when** the final check has failed on more changes than the project's limit over its last hundred landings, **then** abcd restores the full early run for every change, says so with the figures, and only the person can cut it again; the limit is the project's to set, and where it sets none abcd derives it from the project's own measured history and shows the figures it used.
- **Given** a project with one check per change, or with no check that tests the merged result, **when** abcd reads it, **then** it cuts nothing, and says in one sentence what it found.
- **Given** a check that an independent checker can run (itd-2611011325430933), **when** a plan is made, **then** that check appears as one more point in the plan, marked as optional and as never waited for, and it never stands in for the final check; with no checker available the plan says the platform is not covered before landing.
- **Given** a hosted run that fails, **when** abcd reads it, **then** it names the check that failed, whether a rerun of the same commit passed (an intermittent test), and whether that check could have run before the push (an escape); **given** the failures since a stated date, **then** it counts escapes per check.
- **Given** fifty landed changes after the budget is on, **when** the report is read, **then** it gives, per landed change, the plan stated before the push beside the runs the change had, the time from push to landing, the machine time by where it was spent, and the final check's failure rate.

## SOTA

Surveyed from the sources the three sibling intents already read (the [forge, CI and merge note](../../research/notes/2026-10-23-forge-ci-merge-sota.md) and the [local-first landing note](../../research/notes/2026-10-26-local-first-landing-for-private-repos.md)) and from the forge's documentation as read on 2026-11-24.

- **Testing the merged result before it lands (merge queues, merge trains, the bors lineage):** mature. Adopted as the fixed point: the budget never touches that run.
- **Light checks on the proposal, heavy checks before the merge ("two-step CI"):** a named practice, offered as a product feature by one vendor and absent as a built-in from the largest forge's queue. Adopted as the shape; the decision of what is light is made per change from a declaration, not fixed at setup.
- **Selection by dependency (build systems with an affected-targets query):** mature inside those build systems, each a toolchain a project would have to adopt. Not adopted. The declaration's inputs are the seam a project already on one can fill from it.
- **Path filters on workflow triggers:** built into the forges, and unsafe with required checks, because a check that never reports blocks the merge. Not adopted; a check that stands a stage down still reports.
- **Learned test selection:** mature at very large scale and needs a history one project lacks. Not adopted.

**Path 2:** a native floor (one declaration, one classifier, a plan per change) with seams for a build system's own selection and for an independent checker. No new dependency.

## Decisions

- 2026-11-24, the technical facilitator on auto: the budget is its own intent. Each of the three sibling drafts is bounded by scope conditions the budget must cross (one forge and a queue; no queue; one machine), and folding it into one would rewrite a draft already ruled on.
- 2026-11-24, the technical facilitator: safety rests on one rule, recorded as adr-2611241456284353. The plan may only move a check later, never drop it, so no local result has to be trusted for anything.
- 2026-11-24, the product thinker: abcd picks what to save first per project. By default it chooses speed for a public repository and the lowest spend for a private one, says which it picked, and the project can set it.
- 2026-11-24, the technical facilitator, after the product thinker asked why the late-failure limit was a fixed number: the limit is configurable per project. Its default is derived from the project's measured history: the share of changes that would fail the final check if every early test-stage failure moved to it. For abcd's own repository on 2026-11-24 that is about 1 in 12 (19 of the last 300 proposal runs failed in a test stage after a local pass, and 3 of the last 200 final runs failed). A project with fewer than a hundred landings uses 1 in 20 until it has them.
- 2026-11-24, the technical facilitator: counting escapes (hosted failures a check before the push would have caught) moves here from itd-2610261025516272, where the product thinker ruled on 2026-10-26 that they be recorded (dcn-2610261152329451). The ruling stands; the count belongs with the measure.
- 2026-11-24, the technical facilitator: the measured saving is claimed by the intents that cut a run, not here. This intent ships on the plan being right and the final check being whole, so it can be finished before any run is cut.
- 2026-11-24, the technical facilitator: the budget widens by itself and narrows only by the person. Restoring a check can never land a fault; cutting one again is the person's act.
- 2026-11-24, the technical facilitator: the early hosted run is cut by who proposed the change (write access or not), never by a receipt the forge can see, which would be the status adr-2611011327164895 rejects.

## Open Questions

_All resolved; the rulings are in the Decisions section._

1. resolved: what abcd saves first is picked per project, speed for public and lowest spend for private, and is configurable (2026-11-24).
2. resolved: an extra check that has not answered never holds a landing; it is optional and reports afterwards (2026-11-24, adr-2611241456294452).

## Decomposition

| Part | Kind | Home |
|---|---|---|
| The plan per change; the one declaration of checks, inputs and points; the priority; the limit and the restore; the "does nothing" paths | capability | this intent |
| The check of the merged result runs every check for every change and is never cut; earlier runs are warnings, never the grounds for landing | trust rule | adr-2611241456284353, then a brief invariant in this intent's first change |
| Each check runs once, where it is cheapest to trust | stance | the principle `each-check-runs-once-where-it-is-cheapest-to-trust`, written with this intent and cited by itd-2610261025516272 |
| The declaration's schema, the classifier, the measure | plumbing | the spec and the brief |

Verdict: FILE-AS-IS. Typed links: `builds_on` itd-2611011325421304, whose reuse report is the local half of the measure. Reversal flags: none.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the same tests run at three points because no point knows what the others ran; we expect one declaration and a plan per change to let any check move later with none lost, and fifty landings whose stated plans differ from their runs, or one landing without its full final check, would show it wrong
