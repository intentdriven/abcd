---
id: itd-2609201925079472
slug: an-autonomous-implementation-run-paces-itself-by-default-and
spec_id: spc-2609202134341288
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817]
related_intents: [itd-29]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# An autonomous run paces itself by default, and one flag sets the pace

## Press Release

> **A run started with `abcd build` (or `abcd drain`, or the `abcd implement` loop they hand over to) follows a pace it was never told: a working window, a pause after it, and a ceiling on lanes alive at once.** The three numbers come from one place, layered: a flag on this run wins over the repository's abcd configuration, which wins over the machine's, which wins over the bundled default; the run record names which layer applied. `--pace <work-minutes>/<pause-minutes>` and `--sub-agents <n>` set them for one run. At the window's end the loop starts no new lane, lets a running lane finish its step and checkpoint to its branch, writes `next_eligible_at` into the state file and exits; an invocation before that time refuses and names it, so the pause survives a killed process and needs no sleeping model. Today every autonomous run carries its pace in its prompt as prose, and two runs on one machine cannot share it.
>
> "Two pilots, two prompts, two paces typed by hand, and no way to know the other one was keeping to its ceiling," said a technical facilitator who ran both. "Now the pace is a line in the config and a timestamp in the state file, and a run that starts early is refused."

## Why This Matters

Two pilots on one machine in the week of 2026-09-15 ran on two paces written into their prompts by hand, two hours of work then five off with two lanes here, two hours then four off with three lanes there, and neither could be sure the other honoured its ceiling. The pace exists because the model budget resets on a window; a run that ignores it runs into the reset mid-lane and loses the lane. 

## Mechanism

We expect a pace read from layered configuration and enforced by a timestamp in the state file to be honoured by every run without being told, because the loop that reads the state is the only thing that starts a lane; shown wrong if a run starts a lane inside a pause or above the ceiling, which the run record would show, or if the budget window the pause exists for moves in a way minutes cannot express.

## Scope Conditions

- Holds for a run driven by the implement verb's loop; a session pacing itself by prompt is outside it. <!-- cond: cond-2609202134346930 -->
- Holds per run: The ceiling counts this run's lanes and validators. <!-- cond: cond-2609202134341920 -->
- The bundled default is a choice, not a measurement: The two pilots ran 120/300 with two lanes and 120/240 with three, and a repository overrides it in its configuration. <!-- cond: cond-2609202134341625 -->

## What's In Scope

- The three numbers, their four layers, the flags, the window clock and `next_eligible_at` in the state file, the ceiling on this run's lanes and validators, the budget check before a run starts, and the checkpoint on a rate-limit response.

## What's Out of Scope

- A ceiling across runs on one machine: The register's (`itd-2609150819440345`), where a lane registry can live.
- Mid-run telemetry and an operator's hand verbs over the state file: Dropped with `itd-29` until wanted.

## Decisions

Settled on 2026-09-20 with one defensible answer each, on the product thinker's ruling that such a decision is recorded rather than asked (`iss-2609202055103741`):

1. **Re-entrant pause.** The pause is `next_eligible_at` in the state file; an invocation before it refuses and names the time. No process sleeps.
2. **Units are minutes**, as the product thinker specified the flag; a quota-window signal, where a harness exposes one, is a later refinement and is recorded as such.
3. **Cross-run enforcement is the register's.** This intent bounds one run; the design review found a repository file cannot bound two runs across repositories.
4. **Layering is flag, then repository, then machine, then bundled**, the order the model-tier intent already uses.
5. **The bundled default is 120 minutes of work, 300 of pause, two lanes**, the product thinker's numbers for this repository's runs on 2026-09-20; a repository that measured otherwise writes its own.
6. **The budget check and the rate-limit checkpoint come here from `itd-29`**, superseded on 2026-09-20 by the implement verb: A run refuses to start when the estimated cost exceeds the remaining quota where the runner reports one, and a rate-limit response checkpoints the lane and ends the window early.
7. **A run works in parallel up to its ceiling** (ruling DR6, the product thinker, 2026-09-29, verbatim: "per-run agent limit: WORK IN PARALLEL — a run may build several pieces and run reviewers concurrently up to its limit (new build-loop work; then AC6 is testable)."). The validators of a round run side by side, the lanes of steps that do not need each other run side by side, and implementers and reviewers share the ceiling; criterion 6 is tested through the concurrent loop the spec's piece 6 designs.
8. **Steps run one after another unless their plan says otherwise, and a hand-back holds the siblings' landings** (rulings DR6b and DR6c, the product thinker, 2026-09-30). DR6b, verbatim: "(a) ONE AFTER ANOTHER BY DEFAULT: a step runs alongside earlier ones only if its plan says so; nothing already planned changes; reviews run side by side; the 2026-09-21 wording stands." DR6c, verbatim: "(c) FINISH, BUT HOLD THEM: when one piece is handed back, pieces in flight finish but nothing merges until the person re-plans; the person then decides whether the held pieces land as they are." A step's default `needs` is every earlier step, opted out of per step; after a hand-back the sibling lanes run to completion and each passing one is `held`, never armed, until the person releases or discards it with `implement step --release <lane>` or `--discard <lane>` (the spec's piece 6, criteria C6, C11 and C13).

## Open Questions

_None open._

## Acceptance Criteria

- **Given** no flag and no configuration, **when** a run starts, **then** it runs on 120/300 with two lanes and the run record names the bundled layer.
- **Given** a repository configuration and a machine configuration that disagree, **when** a run starts without a flag, **then** the repository's values apply and the record says so.
- **Given** `--pace 90/240 --sub-agents 3`, **when** a run starts, **then** those values apply over every configured layer and the record names the flag.
- **Given** a window that has elapsed, **when** the loop is invoked, **then** it starts no lane, lets a running lane finish its current step and checkpoint to its branch, writes `next_eligible_at`, and exits 0 naming the time.
- **Given** `next_eligible_at` in the future, **when** the loop is invoked, **then** it refuses naming the time and changes no state.
- **Given** the ceiling reached, **when** the loop would start a lane or a validator, **then** it starts nothing, names the lanes alive, and exits; the next invocation fills the slot, and the record counts the minutes the slot was waited for.
- **Given** a runner that reports remaining quota and an estimate that exceeds it, **when** a run starts, **then** it refuses naming both numbers and writes no state; a runner that reports no quota is named and the check is skipped out loud.
- **Given** a rate-limit response from a runner mid-lane, **when** the loop reads it, **then** the lane is checkpointed to its branch, the window ends early with `next_eligible_at` set, and the record names the response.
- **Given** a malformed pace or ceiling, **when** a run starts, **then** it refuses naming the value and the accepted form, and writes no state.

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement verb): The loop this pace bounds.
- **refines `itd-2609170822093401`** (the model tier): The same configuration layering.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-c411be255805 -->
Fidelity review — receipt rcp-c411be255805 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:922986f5804db2851ba4918cd59959f541b9b4df9c2aa7a626ce91f58ab15c81
Input attestations: diff:17490941a92ebc34fcddb14bf7a70c63b61b2b22..45b2057aa51375564f8a0a387b1d60cb84270904 on build/run-2610092052125685-lane-1@sha256:d925a34a08af3bd6325be86da1455a2e646dd87f6e4612769a7509b5a6d8f9ed;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the bundled constants are 120/300/2, a start with no flag and no configuration resolves to them with layer 'bundled', and the run record's pace line names the bundled layer; the test passed on the lane tip
  evidence: internal/core/implement/loop/pace.go:28 — "BundledWorkMinutes = 120"
  evidence: internal/core/implement/loop/pace.go:30 — "BundledSubAgents = 2"
  evidence: internal/core/implement/loop/pace_test.go:69 — "func TestARunWithNoConfigurationRunsOnTheBundledPace"
  evidence: internal/core/implement/loop/pace_test.go:82 — "the record names the pace and the bundled layer"
- ac-2 — MET: resolvePace reads flag, then repository, then machine, then bundled through layered.Load; with the two files disagreeing and no flag the run takes the repository's 100/200/4, the origin is .abcd/config.json and the record names it and not the machine's file
  evidence: internal/core/implement/loop/pace.go:200 — "func resolvePace(roots layered.Roots, f paceFlags) (Pace, error)"
  evidence: internal/core/implement/loop/pace_test.go:96 — "func TestTheRepositoryPaceWinsOverTheMachines"
  evidence: internal/core/implement/loop/pace_test.go:106 — "wantPace(t, st.Pace, 100, 200, 4, "repo")"
- ac-3 — MET: the flags are set on the layered store above every file layer; with both files set, --pace 90/240 --sub-agents 3 resolves to 90/240/3 at layer 'flag' and the record carries the flags as typed
  evidence: internal/core/implement/loop/pace.go:206 — "s.SetFlag(paceNamespace+"."+keyWorkMinutes, *f.work, f.paceOrigin)"
  evidence: internal/core/implement/loop/pace_test.go:134 — "func TestThePaceFlagsWinOverEveryLayer"
  evidence: internal/core/implement/loop/pace_test.go:144 — "st.Pace.WorkMinutes.Origin != "--pace 90/240" || st.Pace.SubAgents.Origin != "--sub-agents 3""
- ac-4 — MET: advance closes an elapsed window before any move: closeWindow writes next_eligible_at and a 'pause' record line, the call returns nil error with the time in Next, the outstanding await's receipt is still taken inside the pause so the lane finishes its implement step on its branch, and the multi-lane test shows nothing starts on any lane
  evidence: internal/core/implement/loop/loop.go:622 — "if until, ok := windowElapsed(*st, now); ok {"
  evidence: internal/core/implement/loop/loop.go:666 — "st.NextEligibleAt = &until"
  evidence: internal/core/implement/loop/pace_test.go:245 — "func TestAnElapsedWindowStartsNothingAndWritesNextEligibleAt"
  evidence: internal/core/implement/loop/pace_test.go:289 — "the running lane checkpoints inside the pause"
  evidence: internal/core/implement/loop/parallel_test.go:738 — "func TestAnElapsedWindowStartsNothingOnAnyLane"
- ac-5 — MET: the first thing advance does under the lock is refuse with a 'pause' contention naming next_eligible_at in RFC3339 when now is before it, returning false so nothing is written; the test compares the state bytes before and after and finds them equal
  evidence: internal/core/implement/loop/loop.go:606 — "if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {"
  evidence: internal/core/implement/loop/loop.go:607 — "the run is paused until "+st.NextEligibleAt.UTC().Format(time.RFC3339)"
  evidence: internal/core/implement/loop/pace_test.go:297 — "a step inside the pause is refused naming the time"
  evidence: internal/core/implement/loop/pace_test.go:300 — "a refused step inside the pause changes no state and performs nothing"
- ac-6 — MET: slotsInUse counts every outstanding await (implementers and validators alike) against the pace's sub_agents; at the ceiling moveAgent hands out nothing, sets CeilingReached, names every lane alive with its awaits, and holds the work in waiting with the time first held; the next receipt frees the slot and tookSlot writes a 'slot' record line counting the minutes waited (14 in the test)
  evidence: internal/core/implement/loop/schedule.go:526 — "if st.slotsInUse() >= st.ceiling() {"
  evidence: internal/core/implement/loop/schedule.go:667 — "took a freed slot after waiting %d minute(s) at the run's ceiling of %d"
  evidence: internal/core/implement/loop/parallel_test.go:300 — "func TestTheCeilingCountsValidatorsAndHoldsTheWaitingWork"
  evidence: internal/core/implement/loop/parallel_test.go:353 — "the intent-auditor of lane-1 took a freed slot after waiting 14 minute(s)"
- ac-7 — MET_WITH_CONCERNS: checkBudget runs in start after the checks and before any directory is made; a route whose quota is below its estimate refuses at stage 'budget' naming the route and both numbers, and the run tier is proven absent; a route reporting none gets a 'reports no quota, so its check is skipped' row in the result, the text and the record. Concern: no shipped runner implements QuotaReporter (quota.go says so and TestOnlyARunnerThatCanReportsAQuota proves host, claude and opencode all report none), so the refusal path is reachable only through the Options.Quota test seam; in production every route is skipped out loud today
  evidence: internal/core/implement/loop/loop.go:328 — "budget, err := checkBudget(len(chk.steps), chk.Intent != "", o)"
  evidence: internal/core/implement/loop/budget.go:131 — "reports %d tokens of quota left, and the run's estimate is %s"
  evidence: internal/core/implement/loop/budget.go:124 — "reports no quota, so its check is skipped (an estimate of %s)"
  evidence: internal/core/implement/loop/budget_test.go:37 — "func TestAnEstimateOverTheQuotaIsRefusedAndWritesNothing"
  evidence: internal/core/implement/loop/budget_test.go:103 — "func TestARouteThatReportsNoQuotaIsSkippedOutLoud"
  evidence: internal/core/runner/quota.go:8 — "No shipped runner does: the adapters read only"
  evidence: internal/surface/cli/build_budget_surface_test.go:65 — "budget: the host reports no quota, so its check is skipped"
- ac-8 — MET_WITH_CONCERNS: both adapters map a rate-limit response to ReasonRateLimited, the loop's dispatcher pauses on it instead of falling back, and rateLimited writes under the lock a 'rate-limit' line naming lane, role, runner and abcd's detail, a 'checkpoint' line per lane in flight naming its branch and head commit, and next_eligible_at once (now plus the run's pause), exiting 0 naming the time; the two-lane test on real worktrees passes. Concerns: (1) 'checkpointed to its branch' is realised as a record line naming the branch and the commit it already holds (checkpointNote); the loop commits nothing, so uncommitted agent work is not checkpointed, which the lane report records as a settled choice; (2) only a routed runner's response is read, so with the default all-host routing the path can never fire; (3) the claude rate_limit_event/rejected and opencode statusCode 429 shapes are stated assumptions not checked against a live harness
  evidence: internal/core/implement/loop/drive.go:152 — "if out.RateLimited != nil {"
  evidence: internal/core/implement/loop/ratelimit.go:60 — "func rateLimited(repoRoot, runID, laneID string, aw Await, fl *runner.Failure, o Options) (StepResult, error)"
  evidence: internal/core/implement/loop/ratelimit.go:89 — "if st.NextEligibleAt == nil || !now.Before(*st.NextEligibleAt) {"
  evidence: internal/core/implement/loop/ratelimit.go:126 — "%s checkpointed to %s at %s, at its %s stage"
  evidence: internal/core/runner/dispatch.go:57 — "PauseOnRateLimit bool"
  evidence: internal/core/runner/claude.go:90 — "the live shape is owed to a person's check"
  evidence: internal/core/implement/loop/ratelimit_test.go:43 — "func TestARateLimitResponseCheckpointsEveryLaneAndEndsTheWindow"
  evidence: internal/core/runner/ratelimit_test.go:19 — "func TestARateLimitResponseIsItsOwnFailure"
- ac-9 — MET: parsePaceFlags refuses a flag that is not two digit runs round one slash, and resolvePace refuses a file value outside its range or a misspelt key, every refusal at stage 'pace' with paceForm as the remedy naming the accepted form; sixteen malformed cases (flags, repository and machine files) each name the value and leave the run tier absent
  evidence: internal/core/implement/loop/pace.go:62 — "--pace takes < work-minutes>/< pause-minutes> in whole minutes (work 1 to 10080, pause 0 to 10080, e.g. 120/300)"
  evidence: internal/core/implement/loop/pace.go:158 — "return f, paceRefusal(fmt.Sprintf("--pace %q is not a pace", layered.BoundKey(*pace)))"
  evidence: internal/core/implement/loop/pace_test.go:155 — "func TestAMalformedPaceIsRefusedAndWritesNoState"
  evidence: internal/core/implement/loop/pace_test.go:203 — "runTierAbsent(t, repo.Root())"

Gap audit:
- honoured:
  - the three numbers come from one layered place (flag, repository, machine, bundled) and the run record names which layer applied
    evidence: internal/core/implement/loop/pace.go:200 — "func resolvePace(roots layered.Roots, f paceFlags) (Pace, error)"
    evidence: internal/core/implement/loop/loop.go:389 — "Note: "pace " + pace.String() + "; the first window opens now""
  - the pause is re-entrant: next_eligible_at in the state file, an early invocation refused naming the time, no process sleeps
    evidence: internal/core/implement/loop/loop.go:606 — "if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {"
    evidence: internal/core/implement/loop/loop.go:666 — "st.NextEligibleAt = &until"
  - the ceiling counts this run's implementers and validators together and a run works in parallel up to it
    evidence: internal/core/implement/loop/schedule.go:68 — "func (s State) slotsInUse() int {"
    evidence: internal/core/implement/loop/parallel_test.go:387 — "func TestAnImplementerAndReviewersShareTheSlots"
  - the budget check runs before anything is written and a refused run leaves no state
    evidence: internal/core/implement/loop/loop.go:326 — "The budget check (criterion 7) comes after the checks and before"
    evidence: internal/core/implement/loop/budget_test.go:47 — "runTierAbsent(t, repo.Root())"
  - a rate-limit response ends the window early for the whole run, next_eligible_at written once, with no fallback to the host and no fallback receipt
    evidence: internal/core/implement/loop/ratelimit.go:89 — "if st.NextEligibleAt == nil || !now.Before(*st.NextEligibleAt) {"
    evidence: internal/core/runner/ratelimit_test.go:60 — "a rate limit is no fallback: outcome %+v, receipts %v"
- diverged:
  - a lane 'checkpointed to its branch' on a rate limit: delivered as a run-record line naming the branch and the commit it already holds; the loop makes no commit, so uncommitted in-flight work stays only in the lane's worktree
    evidence: internal/core/implement/loop/ratelimit.go:15 — "The commits an agent made are on the branch already; the loop makes none of"
    evidence: internal/core/implement/loop/ratelimit.go:126 — "%s checkpointed to %s at %s, at its %s stage"
  - the rate-limit checkpoint applies to any agent mid-lane: delivered only for a role routed to a runner; a host-run agent (the default route) has no way to report a rate limit to the loop
    evidence: internal/core/implement/loop/drive.go:152 — "if out.RateLimited != nil {"
    evidence: internal/core/runner/dispatch.go:53 — "PauseOnRateLimit is set by a caller that pauses its run on a rate-limit"
  - docs/reference/terminology.md now states the run 'checkpoints every lane in flight to its own branch' and resumes from the lanes' branches; the delivered checkpoint is a record line over commits already on the branch, so the sentence overstates what is persisted
    evidence: docs/reference/terminology.md:53 — "it checkpoints every lane in flight to its own branch and writes the time the next window may start"
    evidence: internal/core/implement/loop/ratelimit.go:118 — "func checkpointNote(repoRoot string, l Lane) string {"
- missing:
  - a production quota source: no shipped runner implements QuotaReporter, so the estimate-versus-quota refusal (decision 6) never fires outside the test seam
    evidence: internal/core/runner/quota.go:8 — "No shipped runner does: the adapters read only"
    evidence: internal/core/runner/ratelimit_test.go:97 — "%s reports no quota: ok %v, err %v"
  - a live check of the harness rate-limit event shapes the adapters parse (claude rate_limit_event status 'rejected', opencode error statusCode 429)
    evidence: internal/core/runner/claude.go:90 — "the live shape is owed to a person's check, as the opencode"
    evidence: internal/core/runner/opencode.go:114 — "live shape is owed to a person's check, as its other events' are"

Scope-condition dispositions:
- cond-2609202134346930 — survived: every enforcement point (the pause refusal, the window close, the ceiling, the budget check and the rate-limit checkpoint) lives inside the implement verb's loop and its process driver; nothing reaches a session that paces itself by prompt
  evidence: internal/core/implement/loop/loop.go:606 — "if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {"
  evidence: internal/core/implement/loop/loop.go:328 — "budget, err := checkBudget(len(chk.steps), chk.Intent != "", o)"
  evidence: internal/core/implement/loop/drive.go:145 — "PauseOnRateLimit: true,"
- cond-2609202134341920 — survived: the ceiling is the run's own pace.sub_agents and slotsInUse counts only the awaits in this run's state file, implementers and validators alike; nothing counts another run's agents, and the rate-limit checkpoint frees the slot inside the same run's state
  evidence: internal/core/implement/loop/schedule.go:60 — "func (s State) ceiling() int {"
  evidence: internal/core/implement/loop/schedule.go:68 — "func (s State) slotsInUse() int {"
  evidence: internal/core/implement/loop/parallel_test.go:300 — "func TestTheCeilingCountsValidatorsAndHoldsTheWaitingWork"
- cond-2609202134341625 — survived: the bundled 120/300/2 is three named constants recorded as the product thinker's choice, applied only when no layer sets a key, and a repository's .abcd/config.json pace block overrides each key on its own, as the repository-wins test shows
  evidence: internal/core/implement/loop/pace.go:23 — "The bundled pace: 120 minutes of work, 300 of pause, two lanes (decision 5,"
  evidence: internal/core/implement/loop/pace_test.go:96 — "func TestTheRepositoryPaceWinsOverTheMachines"
<!-- abcd-review-end receipt=rcp-c411be255805 -->

## Grounds

- pursued: we expect a pace read from layered configuration and enforced by a timestamp in the state file to be honoured by every run without being told, because the loop that reads the state is the only thing that starts a lane; shown wrong if a run starts a lane inside a pause or above the ceiling, which the run record would show
