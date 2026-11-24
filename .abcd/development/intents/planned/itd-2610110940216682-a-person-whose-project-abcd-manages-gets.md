---
id: itd-2610110940216682
slug: a-person-whose-project-abcd-manages-gets
spec_id: spc-2611241502250961
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611241456272313]
related_issues: [iss-2610262042399506]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2611241456272313]
---

# Every change gets its full check once, on exactly what lands

## Press Release

> **A project that abcd manages checks each change in full once, at the moment it lands, on exactly the result that will land, and stops paying for the same check two or three times.** Where the project's forge tests a change merged onto the default branch before it lands, that run is the full check and is never skipped. The run on the pull request keeps only the checks that answer in minutes and the ones only a pull request can make; the long test stages wait for the landing. A change from someone outside the project still gets the full check on its pull request. abcd reads how the project's checks are laid out, says in plain words which runs repeat each other and what to change, and says "nothing to do" where there is nothing to remove: a project whose checks take a minute, or one whose forge has no such landing run, keeps what it has. One page tells the person what runs where, and what it means when a change is turned back at the landing.
>
> "My agents finished a fix and then I watched the same hour of tests run three times before it landed," said Maya, who runs a small product with two collaborators. "Now a change is ready for review in minutes, the one full check runs on what actually lands, and when a change is turned back I am told which test did it and whether it was the test's fault."

## Why This Matters

A person who is not a CI engineer wants more checking, not less, and has no way to see when that checking repeats itself. Laid out naively, a careful project checks every change three times: on the contributor's machine before the push, on the pull request, and again when the forge tests the merged result. Only the last of these tests what lands; the first two test a branch that the default branch may already have moved past.

abcd's own repository is the measured case. On 2026-11-24 a change waited through a local check of about 50 minutes, a pull-request run of about an hour and a landing run of 40 to 60 minutes: about two and a half hours of testing for one result, with 195 of the last 200 landing runs passing. In October the same repository ran its full CI 300 times for 71 merged changes and once sat five hours with nothing merging. A second contributor does not add throughput to such a project; they add waiting.

The repeated runs are not free safety. A check that repeats one already made adds delay and cost and finds almost nothing the landing run would not find an hour later, while the landing run is the only one that cannot be replaced: no contributor's own word and no earlier run tests the merged result (adr-2611241456284353). So the full check is spent once per change, where it counts.

## Mechanism

We expect the hosted checking time per landed change to fall by about half, and the time from "ready" to "landed" to fall by about the length of the pull-request run, with no change reaching the default branch without its full check, because the forge's landing run already executes every required gate on the exact merged result: for a change whose contributor already ran the project's checks before pushing, the pull-request run's long test stages repeat work done before and done again after. The saving holds as long as late failures stay rare: a change that fails at the landing costs one landing run for itself and for each change waiting behind it.

Shown wrong if, over at least forty landings in a project that takes this shape, measured against the same project's forty landings before: (a) hosted checking minutes per landed change do not fall by at least a third; or (b) the median time from ready to landed does not fall; or (c) changes fail the landing run at their first attempt more often than the project's own late-failure limit (itd-2611241456272313) and the long stages are not back on the pull request by the next change; or (d) any change reaches the default branch whose merged result did not pass every required gate; or (e) a defect reaches the default branch that the removed pull-request stages would have caught and the landing run did not.

## Scope Conditions

- Holds where the forge tests a change merged onto the current default branch before it lands and can require that run (a merge queue, a merge train). Where no such run exists, the pull-request run stays whole, abcd changes nothing and says so; landing without a forge-run check of the merged result is itd-2610261025516272's ground. <!-- cond: cond-2611241502259911 -->
- Holds for changes pushed by the project's own contributors and their agents after the project's check passed on their machine. A change from outside the project (a fork) keeps the full check on its pull request. <!-- cond: cond-2611241502259564 -->
- Holds where the full check is long enough to matter. Where the whole suite answers in a few minutes abcd reports nothing to gain and proposes nothing. <!-- cond: cond-2611241502257291 -->
- A platform the forge cannot run reliably on every landing (a scarce hosted runner) is checked on a schedule and before each release, not at the landing; an independent checker (itd-2611011325430933) may add an optional check of it per change. "In full" means every gate the default branch requires at a landing. <!-- cond: cond-2611241502257923 -->
- This intent defines the shape, takes it in the one repository that has all three runs, and checks any project against it. Writing or rewriting a project's CI stays with itd-106, whose templates take this shape. <!-- cond: cond-2611241502257967 -->
- Tests that fail intermittently break the saving and are treated as defects: one that turns a change back at the landing is recorded, never retried out of sight. <!-- cond: cond-2611241502259528 -->

## Acceptance Criteria

- **Given** a project whose forge tests the merged result before a landing, **when** a contributor of the project opens or updates a pull request, **then** the pull-request run executes the quick gates and the gates only a pull request can make, the long test stages do not run, and every required check still reports a conclusion that says which stages were left to the landing run.
- **Given** the same project, **when** a change enters the landing run, **then** every gate the default branch requires runs on the merged result, and no setting, earlier run, local result or contributor's statement lets the change land without it.
- **Given** a pull request from outside the project, **when** it is opened or updated, **then** the full check runs on the pull request as before.
- **Given** a project with no forge-run check of the merged result, or one whose full check takes only a few minutes, **when** abcd reads its checks, **then** it reports that nothing is to be changed, and changes nothing.
- **Given** any project abcd manages, **when** the person asks how its checks are laid out, **then** abcd names each run a change goes through, which of them repeat another, and what each repeated run costs per change in minutes, and writes nothing to the project's CI.
- **Given** a change turned back by the landing run, **when** abcd reports it, **then** it names the failing check, and if the same check passes unchanged on the next attempt, abcd records the test as intermittent in the project's issue ledger and puts the change back in line once; a second failure stops and is handed to the person.
- **Given** the project's late-failure limit, set by the project or derived from its own history (itd-2611241456272313), **when** more changes than that fail the landing run at their first attempt, **then** the long test stages return to the pull-request run for every change until the person cuts them again, and abcd says so with the figures.
- **Given** a platform checked on a schedule and not at the landing, **when** that scheduled check last failed, **then** abcd says so before a release is started, not at the release gate.
- **Given** forty landings after the shape is in place, **when** the report is read, **then** it states the hosted minutes per landed change, the time from ready to landed, and the share of changes that passed the landing run first time, each against the forty landings before.
- **Given** the page abcd keeps on the project's checks, **when** the person reads it, **then** it says in plain words what runs where and why, and what being turned back at the landing means.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md).
The merge queue itself was surveyed under
[itd-115](../superseded/itd-115-abcd-managed-repos-merge-prs-without-the.md); this pass
covers the CI layout around it.

- **The lineage tests the merge result and re-tests behind a failure**: Zuul
  tests "each change applied to the tip of the branch exactly as it is going
  to be merged" and re-tests the changes behind a failed one without it;
  bors did the same on a staging branch and was phased out for GitHub's queue
  (its page is undated and links to the July 2023 GA post); GitLab's merge
  train removes a failed merge request and restarts the pipelines behind it,
  but documents a job-level `retry` for jobs that fail intermittently: "If it
  succeeds after a retry, the merge request is not removed from the merge
  train". GitHub's queue has no retry of its own, so a retry there must sit
  inside the job.
- **GitHub's queue as documented**: required checks must trigger on
  `merge_group`, or "The merge will fail as the required status check will
  not be reported"; a queue cannot be enabled on a branch protection rule
  whose branch-name pattern contains a wildcard; the availability statements
  agree (the July 2023 changelog and GitHub's March 2024 account: public
  organisation-owned repositories on GitHub.com and every repository on
  Enterprise Cloud or Server, a personal account's public repository named in
  neither); build concurrency is 1 to 100; a status-check timeout and group
  sizes are ruleset settings; "Require all queue entries to pass required
  checks" decides whether a failing entry may ride a passing group; the queue
  "provides the same benefits as the Require branches to be up to date before
  merging branch protection".
- **A superseded group announces itself only to an App**: the `merge_group`
  webhook's `destroyed` action (reason merged, dequeued or invalidated) is
  delivered to GitHub Apps, not to Actions, which is why a polling sweep is
  the Actions-only route; Actions concurrency cancels only with
  `cancel-in-progress: true` and keeps one pending run per group.
- **"Light on the pull request, heavy in the queue" is a named practice**:
  Mergify's "two-step CI" ("Run fast tests on every PR, and run expensive
  tests only before merge"), listed as something the native queue lacks as a
  built-in (vendor page). GitHub's own monorepo account reports groups of 30
  and a 33% shorter wait and says nothing about trimming the pull-request run
  (vendor blog).
- **The scarce runner is a constant**: five concurrent macOS jobs on Free,
  Pro and Team, 50 on Enterprise; a job may run six hours.
- **A nightly run keeps the cache warm within the rules**: caches unused for
  seven days are evicted, 10 GB per repository, and a pull request can
  restore caches made on the default branch but not on other pull requests.

**Path 1** for the queue and its triggers (the forge's merge queue,
`merge_group`, the ruleset's concurrency and timeout knobs, a scheduled run
for the cache), with the workflow layout as abcd's own scaffold. No Go
dependency: the forge feature is adopted as configuration, so the dependency
gate is not reached. On the open questions the evidence says: turn the
up-to-date rule off only where a queue gates the branch, since the queue
re-tests the merge result, which is the gate iss-172's invariant protects,
and refuse to turn it off anywhere else; on a platform flake the choice is
real, eject and re-queue or one in-job retry of the platform job (GitLab's
documented `retry` keeps a flaky merge request in its train; GitHub's queue
has no retry of its own), and re-queueing is one command either way (`gh pr
merge` adds a pull request whose required checks have passed to the queue,
so re-queueing after an ejection is one command, an inference from the
manual, which says nothing about ejections); size build concurrency below
the macOS limit with the timeout above the measured suite. The independent
fit-challenge runs at plan time.

Read again at planning on 2026-11-24: the open questions this section weighed are settled in the Decisions section. The up-to-date rule is left alone, a change turned back by an intermittent test is put back in line once in the open and never retried inside the job, and the sweep of superseded runs stays a polling workflow.

## Open Questions

_All resolved; the rulings are in the Decisions section._

1. resolved: abcd checks a project's CI against the shape and says what to change; writing a project's CI stays with itd-106 (dcn-2610271112330536, 2026-10-27).
2. resolved: the up-to-date rule is left alone; once the pull-request run takes minutes, keeping a branch up to date costs minutes, and iss-2609012313465609 is measured again after this ships (2026-11-24).
3. resolved: a change turned back by an intermittent test is put back in line once, in the open, and the test is recorded; nothing is retried inside the job (2026-11-24).
4. resolved: a project without a forge-run check of the merged result keeps the full suite on its pull requests (2026-11-24).
5. resolved: the sweep of superseded runs stays a polling workflow (2026-11-24).
6. resolved: how often a late failure may hold others up is a limit each project can set, with a default derived from its own measured history (2026-11-24).

## Decisions

- 2026-10-27, the product thinker (dcn-2610271112330536): itd-106 owns writing a managed repository's CI, and this intent defines the shape. This intent no longer claims to set the workflow up. The edge runs from itd-106 to this intent when that draft is planned, since a template cannot take a shape that does not exist yet; iss-2610262042399506 is resolved by this text.
- 2026-11-04, the product thinker (dcn-2611042202509334): the landing run tests on Linux, and hosted macOS runs nightly, on demand and at each release. The earlier text's "every operating system, in the queue" is withdrawn; "in full" means every gate the default branch requires at a landing.
- 2026-11-24, the technical facilitator on auto: the landing run is the one run never skipped, recorded as adr-2611241456284353, which pays the decision this intent has owed since 2026-10-11.
- 2026-11-24, the technical facilitator: a stage stays on the pull request if it answers in about ten minutes or reads something only a pull request has (dependency review, attribution over the branch's commits, the work-in-progress guard, outside review). The unit and race test stages go to the landing run.
- 2026-11-24, the technical facilitator: a required check that stood a stage down says so and never reports a silent pass; filters on a workflow's trigger are refused, because a check that never reports blocks the merge.
- 2026-11-24, the technical facilitator, after the product thinker asked why the late-failure limit was fixed and not configurable: the limit is per project, with a default derived from the project's measured history (about 1 in 12 for abcd's own repository on 2026-11-24). The limit and the restore are built in itd-2611241456272313; this intent's workflow obeys them.
- 2026-11-24, the technical facilitator: no scarce platform returns to every landing here. What is added is the early warning: a failed scheduled run is reported before a release is started.
- 2026-11-24, the technical facilitator: the queue's build concurrency in abcd's own repository is two (dcn-2611241459375562). It is raised only after the pull-request run has shrunk, by the person on the forge and a new decision.
- Most of the earlier text's plumbing is already built in abcd's own repository and is no longer promised here: no platform's suite on pull requests but the first, no run after every merge, one test pass per platform, and the sweep of superseded queue runs (iss-2610110938282654, iss-2610110938294826, iss-2610110938293273, iss-2610110613372021, all resolved).

## Decomposition

| Part | Kind | Home |
| --- | --- | --- |
| A project's checks are read against the once-per-change shape and the repeats named | capability | this intent |
| Every gate the default branch requires runs on the merged result of every change before it lands, and nothing stands in for that run | trust rule | adr-2611241456284353 |
| The pull-request run of the first project to take the shape drops its long test stages | plumbing | this intent |
| Which checks a change needs and the limit past which an early run returns | capability | itd-2611241456272313 |
| Writing the shape into a project's CI | capability | itd-106 |
| Per-change coverage of a scarce platform | capability | itd-2611011325430933, optional |
| The up-to-date rule beside a queue | measured record | iss-2609012313465609 |

Verdict: FILE-AS-IS, confirmed at planning on 2026-11-24.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: each change waits through the same hour of tests three times and 195 of 200 landing runs pass; we expect hosted minutes per landed change to fall by at least a third with the long stages left to the landing run, and forty landings that show no fall, or late failures above the project's limit, would show it wrong
