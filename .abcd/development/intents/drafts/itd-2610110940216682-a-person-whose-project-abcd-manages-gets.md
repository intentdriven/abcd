---
id: itd-2610110940216682
slug: a-person-whose-project-abcd-manages-gets
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Every change is checked in full once, on exactly what lands

## Press Release

> A person whose project abcd manages gets checks that are thorough and still fast. Each pull request gets the quick checks in minutes. The full suite, on every operating system the project supports, runs once per change, in the merge queue, on exactly what will land, so nothing reaches the default branch unchecked and nothing is checked three times. abcd sets this up when it prepares the repository: the workflow, a clean-up that stops runs the queue has abandoned, a queue sized to the runners the account actually has, a nightly full run in place of a run after every merge, and one test pass per platform. The person never tunes CI by hand, and a small project stops spending most of its CI time repeating itself.

## Why This Matters

abcd is built for people who know what they want and are not CI engineers, which argues for more checking, not less. abcd's own repository showed what more checking costs when it is laid out naively: in the six days to 2026-10-11 it ran its full CI 300 times for 71 merged changes, once on the pull request, once or more in the merge queue, and once again after every merge, each run holding one of the account's five macOS runners for about 40 minutes. On 2026-10-11 nothing merged for five hours while queue entries waited for those runners and timed out. A person whose project abcd manages would meet the same wall sooner, with no one to diagnose it. The same safety net costs a third as much when the full suite runs once, where it matters: on the merge result.

## Mechanism

We expect total CI time per merged change to fall by about half, with no change ever landing untested on any supported platform, because the merge queue already runs the full suite on the exact merge result: the pull-request run's full suite, the post-merge run and the second macOS test pass repeat work the queue run or the local preflight already did. Shown wrong if, after abcd's own CI takes this shape (iss-2610110938282654, iss-2610110938294826, iss-2610110938293273), runs per merged change do not fall, or platform defects reach the default branch that the old pull-request run would have caught. <!-- record-lint: forward-looking -->

## Scope Conditions

- Holds for a repository on GitHub whose default branch is gated by a merge queue; a repository without one keeps the full suite on its pull requests, since nothing else tests the merge result.
- Holds where some runner is scarce (macOS on GitHub's free and team plans: five concurrent jobs); with no scarce runner the queue sizing is moot and the saving is the duplicate runs alone.
- Holds where every push already passes a local preflight on the developer's platform, so the pull-request run's platform pass repeats local work.

## Acceptance Criteria

- **Given** a repository abcd prepares that gates its default branch through a merge queue, **when** its CI is set up, **then** a pull request runs the quick checks on one platform, every required check still reports a conclusion on the pull request, and the full suite runs on every supported platform in the merge queue on the merge result.
- **Given** the merge queue rebuilds its groups, **when** a superseded group's runs are still queued or running, **then** they are cancelled, and a run whose queue entry is still live is never cancelled.
- **Given** a merge to the default branch, **when** it lands, **then** no second full run starts for it, and a scheduled full run on the default branch checks it within a day and keeps the build cache warm.
- **Given** the account's concurrency for its scarcest runner, **when** the queue is configured, **then** it builds no more entries at once than that runner allows with room to spare, and its check timeout exceeds the full suite's measured duration.
- **Given** a platform whose suite runs a race-enabled pass, **when** the full suite runs there, **then** it runs one pass, not a plain pass and then a race pass.
- **Given** the checks abcd sets up, **when** the person reads them, **then** one page says in plain words what runs where and why, and what a queue ejection means.

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

## Open Questions

- **Write the workflow, or check the person's own.** The scaffold reads a managed repository's CI and never writes it (it derives check names and writes nothing hard-coded). This intent either lays down a CI workflow of its own, or checks the person's CI against the shape and says what to change.
- **The up-to-date rule.** Whether abcd turns off "require branches to be up to date" where a merge queue gates the default branch; the queue gives the same guarantee, and abcd's own repository kept the rule on 2026-09-01 (iss-2609012313465609 carries the measurement).
- **Where a platform flake lands.** Running a platform's suite only in the queue turns a flake on that platform into a queue ejection and a re-queue (iss-2610051434156079 is one); whether that cost is accepted, or the queue retries a platform job once.
- (SOTA 2026-10-23) Whether a managed repository without a merge queue (a
  personal account's public repository, which neither availability statement
  names) keeps the full suite on its pull requests, as the first scope
  condition says, and how the probe in itd-2610161647248332 tells the two
  apart.
- (SOTA 2026-10-23) Whether the superseded-group sweep stays a polling
  workflow or moves to a GitHub App that receives `merge_group` `destroyed`,
  once a managed repository has an App for other reasons (H2).

## Decomposition

| Part | Kind | Home |
| --- | --- | --- |
| A managed repository's CI runs the full suite once, in the queue | capability | this intent |
| The full suite must run on the merge result of every change before it lands | trust rule | a decision record and a brief invariant, owed before planning |
| A remedy is filed whole, built where it can be, measured where it cannot | stance | `a-remedy-is-filed-whole` (principle) |
| abcd's own CI takes the shape first | plumbing | iss-2610110938282654, iss-2610110938294826, iss-2610110938293273 (built on `ci/full-suite-once-per-merge`), iss-2610110613372021 (superseded queue runs) <!-- record-lint: forward-looking --> |
| Parts that cannot be built yet | measured records | iss-2609012313465609 (up-to-date rule), iss-2610110939547495 (raise the macOS limit), iss-2610110939559446 (receipt as a status) |

Verdict: FILE-AS-IS, the routing awaiting the product thinker's confirmation.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
