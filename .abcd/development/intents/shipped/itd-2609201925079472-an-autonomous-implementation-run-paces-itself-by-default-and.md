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
Input attestations: diff:3f21f58b5d869ea6ed7299b6ad5c38a32a468f6f..8a08de576cdaa182b11719116cb64e43f5bcbc7a on build/run-2610100550297632-lane-1@sha256:73677a02544e3a04a092b71888d0014e06226aefae31de36621b3d27d78320d0;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The bundled constants are 120/300/2 (pace.go:28-30); TestARunWithNoConfigurationRunsOnTheBundledPace asserts the state and the run record name 120/300, 2 sub-agents and the bundled layer, and passes on a scratch copy of 8a08de576; the CLI test asserts the same line in implement status.
  evidence: internal/core/implement/loop/pace.go:28 — "BundledWorkMinutes = 120"
  evidence: internal/core/implement/loop/pace_test.go:82 — "the record names the pace and the bundled layer"
  evidence: internal/surface/cli/build_pace_surface_test.go:61 — "pace: 120/300 minutes, 2 sub-agents (bundled)"
- ac-2 — MET: resolvePace reads each key through layered.Load (flag, repo, machine, bundled); TestTheRepositoryPaceWinsOverTheMachines sets disagreeing machine and repo files, asserts 100/200/4 from layer repo with origin .abcd/config.json, and that the record names the repository's file and not the machine's; it passes.
  evidence: internal/core/implement/loop/pace.go:200 — "func resolvePace(roots layered.Roots, f paceFlags) (Pace, error)"
  evidence: internal/core/implement/loop/pace_test.go:105 — "wantPace(t, st.Pace, 100, 200, 4, "repo")"
  evidence: internal/core/implement/loop/pace_test.go:110 — "the record names the repository's file and not the machine's"
- ac-3 — MET: TestThePaceFlagsWinOverEveryLayer starts with both files set and --pace 90/240 --sub-agents 3, asserts 90/240/3 from layer flag with origins '--pace 90/240' and '--sub-agents 3' and the record naming both flags; the CLI test asserts the rendered line names the flag layer for each value; both pass.
  evidence: internal/core/implement/loop/pace_test.go:143 — "wantPace(t, st.Pace, 90, 240, 3, "flag")"
  evidence: internal/core/implement/loop/pace_test.go:144 — "st.Pace.WorkMinutes.Origin != "--pace 90/240" || st.Pace.SubAgents.Origin != "--sub-agents 3""
  evidence: internal/surface/cli/build_pace_surface_test.go:85 — "pace: 90/240 minutes, 3 sub-agents (work from --pace 90/240, the flag layer"
- ac-4 — MET: advance closes an elapsed window before any move (windowElapsed/closeWindow write next_eligible_at and the pause entry, the call returns nil error with Next naming the time); the pace test verifies the implementer's receipt is still taken inside the pause and the lane advances, and C11 verifies two lanes' receipts both verify with next_eligible_at written once.
  evidence: internal/core/implement/loop/loop.go:647 — "if until, ok := windowElapsed(*st, now); ok {"
  evidence: internal/core/implement/loop/loop.go:708 — "st.NextEligibleAt = &until"
  evidence: internal/core/implement/loop/pace_test.go:293 — "the running lane checkpoints inside the pause"
  evidence: internal/core/implement/loop/parallel_test.go:757 — "both receipts are verified during the pause"
- ac-5 — MET: advance refuses with a 'pause' contention naming next_eligible_at in RFC3339 before any state change; TestAPauseRefusesUntilNextEligibleAt asserts the refusal names 2026-09-25T13:00:00Z and no stage ran, and the pace test asserts the state bytes are identical after the refusal.
  evidence: internal/core/implement/loop/loop.go:632 — "return false, contend("pause", "", "", "the run is paused until "+st.NextEligibleAt.UTC().Format(time.RFC3339),"
  evidence: internal/core/implement/loop/loop_test.go:673 — "if r.Stage != "pause" || !r.Contention || !strings.Contains(r.Reason, "2026-09-25T13:00:00Z")"
  evidence: internal/core/implement/loop/pace_test.go:304 — "a refused step inside the pause changes no state and performs nothing"
- ac-6 — MET: moveAgent at slotsInUse >= ceiling hands out nothing, sets CeilingReached and names the lanes alive; tookSlot records the whole minutes waited when the next call serves the item. C1-C3 (TestTheCeilingCountsValidatorsAndHoldsTheWaitingWork) assert the names, the unchanged waiting time, the slot filled by the next call and the record line '14 minute(s)'; it passes.
  evidence: internal/core/implement/loop/schedule.go:538 — "if st.slotsInUse() >= st.ceiling() {"
  evidence: internal/core/implement/loop/schedule.go:686 — "took a freed slot after waiting %d minute(s) at the run's ceiling of %d"
  evidence: internal/core/implement/loop/parallel_test.go:356 — "the intent-auditor of lane-1 took a freed slot after waiting 14 minute(s)"
- ac-7 — MET_WITH_CONCERNS: budgetCheck runs after every other check and before the run directory is made; an estimate over a reported quota fails the row naming both numbers (TestAQuotaUnderTheEstimateRefusesTheStartNamingBoth asserts '5 agent run(s) left', 'estimated to start 6' and zero runs on disk); a runner reporting none is named and the row says 'skipped' in the checks and the run record. Concerns: (1) the comparison is reached only through the Options.Quota test seam, because QuotaReporter is implemented by no shipped runner (runner.go:135-137), so today every real run skips the check; (2) the estimate is a lower bound that omits fix rounds and syncs (budget.go:13).
  evidence: internal/core/implement/loop/loop.go:332 — "budget := budgetCheck(chk, o)"
  evidence: internal/core/implement/loop/budget.go:133 — "Detail: "the run's estimate exceeds the quota its runner reports: ""
  evidence: internal/core/implement/loop/budget_test.go:79 — "a refused budget writes no state"
  evidence: internal/core/implement/loop/budget_test.go:112 — "func TestARunnerThatReportsNoQuotaIsNamedAndTheCheckSkipped"
  evidence: internal/core/runner/runner.go:135 — "QuotaReporter is a Runner that reports its remaining quota. Neither shipped"
- ac-8 — MET_WITH_CONCERNS: The claude runner reads a rejected rate_limit_event or an assistant rate_limit error as ReasonRateLimited; Dispatch returns it without fallback; Drive routes it to rateLimitWindow, which writes next_eligible_at once (a running pause kept), records the response with the lane and runner, and checkpoints the hit lane; the end-to-end test through Drive and the fake claude asserts all of it and passes. Concerns: (1) 'checkpointed to its branch' is delivered as save-aside plus reset to the branch's last commit, so the agent's uncommitted work lives in the run's aside directory, not on the branch; (2) only the claude CLI's response shapes are recognised; an opencode limit still falls back as an ordinary refusal (DECISIONS.md:2680); (3) a limit met by a host-run agent never reaches the loop (drive.go:152-158 sees only Dispatch failures).
  evidence: internal/core/runner/claude.go:101 — "func claudeRateLimit(out []byte) string {"
  evidence: internal/core/runner/dispatch.go:115 — "if fl.Reason == ReasonRateLimited {"
  evidence: internal/core/implement/loop/drive.go:158 — "return rateLimitWindow(repoRoot, runID, res.Lane, aw, fl, o)"
  evidence: internal/core/implement/loop/ratelimit.go:123 — "st.NextEligibleAt = until"
  evidence: internal/core/implement/loop/ratelimit_test.go:93 — "the record names the response and the lane it came from"
  evidence: internal/core/implement/loop/ratelimit.go:15 — "uncommitted work is kept for review and never built on), and its await"
- ac-9 — MET: parsePaceFlags and the resolver's range check refuse at stage pace naming the value and paceForm (the accepted form); TestAMalformedPaceIsRefusedAndWritesNoState covers 16 malformed flag and file cases and asserts the value, the form and an absent run tier; the CLI test asserts exit 2 with the form named; both pass.
  evidence: internal/core/implement/loop/pace.go:62 — "var paceForm = "--pace takes < work-minutes>/< pause-minutes> in whole minutes"
  evidence: internal/core/implement/loop/pace_test.go:202 — "runTierAbsent(t, repo.Root())"
  evidence: internal/surface/cli/build_pace_surface_test.go:111 — "if code != 2 || !strings.Contains(errOut, "refused at pace") || !strings.Contains(errOut, "< work-minutes>/< pause-minutes>")"

Gap audit:
- honoured:
  - The three numbers come from one place, layered flag over repository over machine over bundled, and the run record names which layer applied
    evidence: internal/core/implement/loop/pace.go:200 — "func resolvePace(roots layered.Roots, f paceFlags) (Pace, error)"
    evidence: internal/core/implement/loop/pace_test.go:147 — "the record names the flags"
  - At the window's end the loop starts no new lane, writes next_eligible_at and exits; an invocation before that time refuses and names it, so the pause survives a killed process
    evidence: internal/core/implement/loop/loop.go:631 — "if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {"
    evidence: internal/core/implement/loop/loop.go:707 — "func closeWindow(st *State, now, until time.Time) {"
  - A ceiling on this run's lanes and validators, with the minutes a slot was waited for counted in the record
    evidence: internal/core/implement/loop/schedule.go:671 — "func tookSlot(st *State, key, role string, now time.Time) {"
    evidence: internal/core/implement/loop/parallel_test.go:321 — "if st.SlotsInUse() != 2 || st.Ceiling() != 2 {"
  - A run refuses to start when the estimated cost exceeds the remaining quota where the runner reports one, naming both numbers and writing no state
    evidence: internal/core/implement/loop/budget.go:126 — "if s.runs > q.Remaining {"
    evidence: internal/core/implement/loop/loop.go:334 — "if !budget.OK {"
  - A rate-limit response is never fallen back on; it ends the window early with next_eligible_at written once and the record naming the response, the runner and the lane
    evidence: internal/core/runner/dispatch.go:115 — "if fl.Reason == ReasonRateLimited {"
    evidence: internal/core/implement/loop/ratelimit.go:116 — "if until == nil || !now.Before(*until) {"
    evidence: internal/core/implement/loop/ratelimit_test.go:185 — "next_eligible_at is written once"
  - The budget line and the rate-limit block are wired on the CLI (build, build next, drain) and the plugin surface
    evidence: internal/surface/cli/drain.go:132 — "cfg, err := loadRunners(cmd, roots)"
    evidence: internal/surface/cli/build_pace_surface_test.go:190 — "func TestBuildNamesTheBudgetCheckAndTheRunnersItSkipped"
    evidence: commands/build.md:180 — "Last, once every other check passes, the budget check asks each runner a role"
    evidence: commands/implement.md:288 — "A runner's rate-limit response ends the window early in the same way"
- diverged:
  - The rate-limited lane is 'checkpointed to its branch': delivered as the agent's uncommitted work and partial receipt saved aside under the run directory and the worktree reset to the branch's last commit, so nothing new reaches the branch
    evidence: internal/core/implement/loop/ratelimit.go:157 — "aside, err := saveAside(repoRoot, root, st.RunID, *lane, lane.Awaits[k], why, aw.Role == RoleImplementer, now)"
    evidence: internal/core/implement/loop/ratelimit_test.go:108 — "lane-2's worktree is reset to its last commit"
    evidence: .abcd/work/DECISIONS.md:2680 — ""Checkpointed to its branch" reads as the restart of 2026-10-09 reads a gone agent"
  - The budget comparison exists only behind an optional QuotaReporter that no shipped runner implements, so every real run today skips the check out loud; the comparison path is exercised through the Options.Quota test seam
    evidence: internal/core/runner/runner.go:139 — "type QuotaReporter interface {"
    evidence: internal/core/implement/loop/budget.go:139 — "detail = "skipped, as no runner of the run reports a quota: " + detail"
    evidence: internal/core/implement/loop/loop.go:57 — "Quota func(name string) (runner.Quota, bool, error)"
  - The estimate from the spec's size is a lower bound: fix rounds and syncs are not counted
    evidence: internal/core/implement/loop/budget.go:13 — "Fix rounds and syncs"
- missing:
  - Rate-limit detection for the opencode runner: an opencode limit is still an ordinary refusal and falls back as before
    evidence: .abcd/work/DECISIONS.md:2680 — "an opencode limit stays a refusal and falls back as before"
    evidence: internal/core/runner/claude.go:80 — "if what := claudeRateLimit(res.stdout); what != "" {"
  - A rate limit met by a host-run sub-agent: the loop reads only a Dispatch failure, and no verb lets the host report one
    evidence: internal/core/implement/loop/drive.go:152 — "out, err := d.Dispatch(ctx, req)"

Scope-condition dispositions:
- cond-2609202134346930 — survived: Every enforcement point is inside the implement verb's loop: the pause and window close in advance, the ceiling in moveAgent, the budget check in start, and the rate-limit checkpoint reached only from Drive's Dispatch; a session pacing itself by prompt, and a host-run agent's limit, are outside what the loop can see, as the condition assumed.
  evidence: internal/core/implement/loop/loop.go:639 — "The window clock (itd-2609201925079472) is the run's, one for every"
  evidence: internal/core/implement/loop/drive.go:158 — "return rateLimitWindow(repoRoot, runID, res.Lane, aw, fl, o)"
- cond-2609202134341920 — survived: The ceiling is this run's pace.sub_agents counted against the awaits in this run's state file (implementers and validators alike), the budget estimate is per run, and a rate limit pauses only the run it hit (the implementer's report notes a drain's own window does not end early), so the per-run assumption held.
  evidence: internal/core/implement/loop/schedule.go:538 — "if st.slotsInUse() >= st.ceiling() {"
  evidence: internal/core/implement/loop/parallel_test.go:314 — "r.Slots != 2 || r.Ceiling != 2"
  evidence: internal/core/implement/loop/ratelimit.go:81 — "func rateLimitWindow(repoRoot, runID, laneID string, aw Await, fl *runner.Failure, o Options) (StepResult, error) {"
- cond-2609202134341625 — survived: The bundled 120/300/2 is three named constants with the comment that a repository that measured otherwise writes its own under pace in .abcd/config.json; the repository override is tested, and the rate-limit pause falls to BundledPauseMinutes only when the run carries no pace.
  evidence: internal/core/implement/loop/pace.go:23 — "The bundled pace: 120 minutes of work, 300 of pause, two lanes (decision 5,"
  evidence: internal/core/implement/loop/pace_test.go:99 — "repoConfig(t, repo, `{"pace": {"work_minutes": 100, "pause_minutes": 200, "sub_agents": 4}}`)"
  evidence: internal/core/implement/loop/ratelimit.go:117 — "pause := BundledPauseMinutes"
<!-- abcd-review-end receipt=rcp-c411be255805 -->

## Grounds

- pursued: we expect a pace read from layered configuration and enforced by a timestamp in the state file to be honoured by every run without being told, because the loop that reads the state is the only thing that starts a lane; shown wrong if a run starts a lane inside a pause or above the ceiling, which the run record would show
