---
id: spc-2611241502250961
slug: a-person-whose-project-abcd-manages-gets
intent: itd-2610110940216682
origin: researcher-authored
production_mode: hand-written
---
# Every change gets its full check once, on exactly what lands

## Summary

Delivers itd-2610110940216682. Where a project's forge tests the merged result before a landing, that run is the full check (adr-2611241456284353), and the run on the pull request keeps only the gates that answer in minutes and the ones only a pull request can make. This spec does three things. It reads any project's checks against that shape and names the runs that repeat each other, or says there is nothing to change. It gives the shape to the one repository that has all three runs, abcd's own, by standing the unit and race test stages down on a pull request from the project's own branches while every required check still reports. And it handles what the shape makes more likely: a change turned back at the landing is named with its failing check, an intermittent test is recorded and the change put back in line once, and a failed scheduled run is reported before a release is started. It builds on the declaration, the classifier, the measure and the late-failure limit of itd-2611241456272313 and writes no project's CI but abcd's own; writing a managed project's CI stays with itd-106. Five steps, the last a measured close after forty landings.

## Approach

### What exists today

Most of the earlier draft's plumbing is built in abcd's own repository: no second platform's suite on pull requests, no run after every merge, one test pass per platform, and `queue-sweep.yml` cancelling superseded queue runs. What is left is the headline. The `check` job in `.github/workflows/ci.yml` still runs "Test (outside internal)", "Test (plain-only guards)" and "Test (race, internal)" on a pull request, about an hour, and the queue runs them again. The `changes` job already classes a pull request as inert or not and treats every other event as never inert, so a queue entry runs every lane; `check (macos-latest)` already reports from a cheap runner on events where its suite does not run and prints that no suite ran. Both are the patterns this spec extends.

### The layout read and the page (step 1)

`abcd checks` with no operand reads the declaration (itd-2611241456272313), the landing rule (`landing.ReadMergeRule`, which already reads whether a merge queue gates the default branch from the ruleset mirror) and the measured run history, and prints each run a change goes through, which runs repeat another in full, and what each repeat costs per change in minutes. It says "nothing to change" in one sentence in three cases: no forge-run check of the merged result, a full check that answers in under ten minutes, and a forge abcd cannot read (the safe answer, since nothing then changes). It writes nothing to a project's CI. `abcd checks page` renders `.abcd/work/checks.md` from the same reading: what runs where and why, and what being turned back at the landing means, in plain words. A lint rule fails when the committed page differs from what the declaration renders.

### abcd's own pull-request run (step 2)

A stage stays on the pull request if it answers in about ten minutes or reads something only a pull request has: build, vet, the format gate, the record and docs gates, the secret scan, smoke, the reading evals, dependency review, attribution over the branch's commits, the work-in-progress guard and outside review. The unit and race test stages run at the landing only.

The `changes` job gains one more output, `long_stages`, which is `run` unless all of these hold: the event is a pull request, its head branch lives in the repository itself (the forge states this; nothing the author sends is read), and the default branch's declaration says `early_run: quick`. The three test steps are conditioned on it. Every other event, the queue entry included, gives `run`, as the inert classification already does. A job never gains a trigger filter: the steps stand down inside jobs that still report, and the `check` job's summary says "left to the landing run: unit, race" so no required check reports a silent pass. With `early_run: full` (the restore of itd-2611241456272313) the stages run on every pull request again, with no workflow change.

The workflow's shape tests are extended first and watched fail: a test reads `ci.yml` and fails unless each test step runs unconditionally on the queue event, and a second fails unless the pull-request condition names all three of its terms. A trial pull request from a branch and one from a fork are opened before the change is relied on. The fourth scope condition of itd-2611011325421304, which names the pull-request run among the places the full uncached check runs, is reworded in the same change to the landing run and the scheduled run, and AGENTS.md's account of CI follows.

Guard on the risk: local checks run on the contributor's own platform and the landing run is Linux, so after this step the landing run is the first Linux test run a change meets. Before the step lands, the measure of itd-2611241456272313 counts how many of the last forty changes' pull-request runs failed in the two stages after a passing local check; if that is above the project's late-failure limit, a plain unit pass (not the race pass) stays on the pull request and the Decisions section of the intent records it.

### Turned back at the landing (step 3)

When the forge removes a change from its landing run, `abcd land` (resumed, or run by the loop that queued the change) reads the failed run and names the failing check. It then looks for the same check passing on the same commits: if the failing check passes unchanged on the next attempt, the test is intermittent. abcd files it in the project's issue ledger (one record per test, a recurrence added to an open one) and puts the change back in line once. A second failure stops, and the change is handed to the person with both runs named. Nothing is retried inside a job, because a retry there hides the failure in a passing conclusion.

### A failed scheduled run before a release (step 4)

`claim`'s scheduled-run reader already knows the last conclusion of the workflow `status.scheduled_workflow` names. Reserving a release (`abcd reserve rel-<version>`) and the release dry run now both print it when that conclusion is a failure, naming the run and the platform, before anything is held. It is a warning and not a refusal: the release gates still decide.

### The measured close (step 5)

Before step 2 lands, the forty landings before are measured and recorded. After forty landings in the new shape the same figures are recorded beside them: hosted minutes per landed change, time from ready to landed, and the share of changes that passed the landing run at their first attempt. iss-2609012313465609 (the up-to-date rule) is measured again on those figures and closed or kept. Raising the queue's build concurrency above two (dcn-2611241459375562) is considered only now, as the person's act on the forge with a new decision and the mirror following.

## How the Acceptance Criteria are met

1. Quick gates on the pull request, long stages left to the landing, every required check reporting what was left: step 2.
2. Every required gate on the merged result, with nothing standing in: the queue event never stands a stage down, held by the shape test (step 2) and by adr-2611241456284353.
3. A pull request from outside the project runs in full: the head-branch term of the condition (step 2), shown by the trial pull request from a fork.
4. No forge-run check of the merged result, or a short suite, changes nothing: the three "nothing to change" results (step 1).
5. The runs named, the repeats and their cost per change, nothing written: `abcd checks` (step 1).
6. The failing check named, an intermittent test recorded and the change put back once, a second failure handed over: step 3.
7. Past the project's limit the long stages return until the person cuts them: the `early_run` term (step 2) with the restore of itd-2611241456272313.
8. A failed scheduled run said before a release is started: step 4.
9. The forty-landing figures against the forty before: step 5.
10. The page in plain words: `abcd checks page` (step 1).

## Open Questions

1. resolved: this spec checks and reports, and edits only abcd's own workflow by hand; it renders no workflow for a managed project (dcn-2610271112330536).
2. resolved: the hosted run learns the restore from the default branch's declaration, which a hosted run can read with no credential; no forge variable and no label carries it.
3. resolved: stages stand down inside jobs that still report; trigger filters are refused.
4. resolved: one re-queue in the open for an intermittent failure, none inside the job.
5. resolved: the failed scheduled run is a warning before the release starts, not a new gate.

## Footprint

- packages: internal/core/checks/layout.go, internal/core/checks/page.go, internal/core/lint/checkspage.go, internal/core/lint/config.go, internal/core/landing/turnedback.go, internal/core/landing/land.go, internal/core/landing/prstate.go, internal/core/claim/scheduled.go, internal/core/launch/preflight_report.go, internal/ghcontract/graphql-fields.txt, internal/surface/cli/checks.go, internal/surface/cli/land.go, internal/surface/cli/reserve.go, commands/checks.md, commands/land.md, commands/reserve.md, commands/launch.md, docs/reference/cli/commands.md, docs/reference/cli/commands/checks.json, .github/workflows/ci.yml, .abcd/checks.json, .abcd/work/checks.md, AGENTS.md, .abcd/development/intents/planned/itd-2611011325421304-a-landing-runs-the-tests-its-change-can.md, .abcd/development/brief/04-surfaces/43-checks.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/37-reserve.md, .abcd/development/brief/04-surfaces/04-launch.md, .abcd/development/research/notes/full-check-once-forty-landings.md, .abcd/work/rulesets/main-protection.json
- tests: internal/core/checks/layout_test.go, internal/core/checks/page_test.go, internal/core/lint/checkspage_test.go, internal/core/lint/cishards_test.go, internal/core/lint/plainguards_test.go, internal/core/lint/prstages_test.go, internal/core/landing/turnedback_test.go, internal/core/claim/scheduled_test.go, internal/core/launch/preflight_report_test.go, internal/surface/cli/checks_surface_test.go, internal/surface/cli/land_surface_test.go, internal/surface/cli/reserve_surface_test.go

## Steps

1. The layout read and the page
   - criteria: 4, 5, 10
   - needs: none
   - packages: internal/core/checks/layout.go, internal/core/checks/page.go, internal/core/lint/checkspage.go, internal/core/lint/config.go, internal/surface/cli/checks.go, commands/checks.md, docs/reference/cli/commands.md, docs/reference/cli/commands/checks.json, .abcd/work/checks.md, .abcd/development/brief/04-surfaces/43-checks.md
   - tests: internal/core/checks/layout_test.go: a project with three runs names each, marks the two that repeat the landing run and gives each repeat's minutes per change from a fixture history; no forge-run check of the merged result, a nine-minute suite and an unreadable forge each give one sentence and no proposal; nothing under the project's CI directory is written in any case. internal/core/checks/page_test.go: the page names what runs where and what being turned back means, from the declaration alone. internal/core/lint/checkspage_test.go: a committed page that differs from the rendering fails, watched fail first. internal/surface/cli/checks_surface_test.go: bare `checks` and `checks page` on the CLI, with `--json`.
2. The pull-request run drops its long test stages
   - criteria: 1, 2, 3, 7
   - needs: 1
   - packages: .github/workflows/ci.yml, .abcd/checks.json, .abcd/work/checks.md, AGENTS.md, .abcd/development/intents/planned/itd-2611011325421304-a-landing-runs-the-tests-its-change-can.md
   - tests: internal/core/lint/prstages_test.go: reading `ci.yml`, each of the three test steps runs unconditionally on the queue event, on the schedule and on demand, watched fail against a condition that also matches the queue event; the pull-request condition stands a step down only for a head branch in the repository with `early_run: quick`, and runs it for a fork and for `early_run: full`; no job carries a path or event filter on its trigger; the `check` job's summary names the stages left to the landing. internal/core/lint/cishards_test.go and internal/core/lint/plainguards_test.go: the shard and plain-guard parity they hold today still holds for the queue event. A trial pull request from a branch and one from a fork, their runs recorded in the pull request that lands this step.
3. A change turned back at the landing
   - criteria: 6
   - needs: none
   - packages: internal/core/landing/turnedback.go, internal/core/landing/land.go, internal/core/landing/prstate.go, internal/ghcontract/graphql-fields.txt, internal/surface/cli/land.go, commands/land.md, .abcd/development/brief/04-surfaces/38-land.md
   - tests: internal/core/landing/turnedback_test.go: from fixtures of the forge's reply, a change removed from the landing run names its failing check; the same check passing unchanged on the next attempt files one issue for the test and re-queues the change once; a recurrence adds to the open issue and files no second; a second failure stops with both runs named and re-queues nothing; a failure with no later pass files nothing and stops. internal/surface/cli/land_surface_test.go: the result and `--json` carry the failing check and the re-queue.
4. A failed scheduled run is reported before a release starts
   - criteria: 8
   - needs: none
   - packages: internal/core/claim/scheduled.go, internal/core/launch/preflight_report.go, internal/surface/cli/reserve.go, commands/reserve.md, commands/launch.md, .abcd/development/brief/04-surfaces/37-reserve.md, .abcd/development/brief/04-surfaces/04-launch.md
   - tests: internal/core/claim/scheduled_test.go: a failed last scheduled run is returned with its run and platform, a passed or unread one with none. internal/surface/cli/reserve_surface_test.go: reserving a release prints the warning before the hold is taken and still takes it. internal/core/launch/preflight_report_test.go: the dry run's report carries the same line.
5. The measured close
   - criteria: 9
   - needs: 2
   - packages: .abcd/development/research/notes/full-check-once-forty-landings.md, .abcd/work/rulesets/main-protection.json
   - tests: the research note's two sets of forty landings, read with `abcd checks measure`, with each of the Mechanism's five falsifiers answered from them

## Size

Five pull requests. Step 1 is a reader and a renderer over what itd-2611241456272313 built. Step 2 is the smallest diff and the riskiest change in the family: it alters what the required checks do on a pull request, and both ways of getting it wrong are silent. A required check that stops reporting blocks every merge until someone notices, and a condition that also matches the queue event lets a merged tree through with its tests skipped and a passing result, which is the worst outcome the intent can produce; the shape test watched fail and the two trial pull requests are the guard. Step 3 touches `land`'s reading of a pull request's state and files into the issue ledger. Step 4 is a few lines over an existing reader. Step 5 is a note and, only if the figures support it, a decision about the queue. Steps 1, 3 and 4 are independent of each other. No step runs a preflight in this planning lane; each step's builder runs the repository's definition of done.
