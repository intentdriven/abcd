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

<!-- abcd-review: INGESTED receipt=rcp-d0980beeafe9 -->
Fidelity review — receipt rcp-d0980beeafe9 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:c03d90ef129362ad3ea7a4100ab7a87495cce96e9da529689d67fc4334ee10af
Input attestations: diff:de3ba5fa..6438860b (PR #662, merged bad1c73e; tree read at cede78b8)@-;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Join is an exclusive create of the session record plus one session_open line; nothing signals or blocks the first session, and the test joins a second session with the first never consulted
  evidence: internal/core/implement/session.go:113 — "Join records a session in the run and logs its session_open. Nothing signals any other session"
  evidence: internal/core/implement/session.go:142 — "fsutil.CreateExclusiveIn(root, sessionRel(id)"
  evidence: internal/core/implement/session_test.go:15 — "func TestJoiningNeedsNoWordFromTheFirstSession"
- ac-2 — MET_WITH_CONCERNS: SetMode writes a window_mode line naming the mode and the window; collisions are claim_denied lines the verb writes; but no window-end line exists, and the lanes each session took are hand-logged lane_open lines tallied per window only by Compare, not carried in the log at the window's end
  evidence: internal/core/implement/session.go:334 — "fields := map[string]any{"mode": string(m)}"
  evidence: internal/core/implement/session_test.go:64 — "func TestAWindowNamesItsMode"
  evidence: internal/core/implement/claim.go:224 — "r.append(req.Session, EventClaimDenied"
  evidence: internal/core/implement/log.go:66 — "loggableEvents are the events `implement log` writes on a session's word"
  evidence: internal/core/implement/report.go:157 — "for _, e := range sorted {"
- ac-3 — MET: the claim is an exclusive file create, a held claim returns HeldError and logs claim_denied naming the holder, and a lease past its expiry is logged claim_lapsed, removed and re-claimed; both paths are test-held including a concurrent-writers test
  evidence: internal/core/implement/claim.go:170 — "fsutil.CreateExclusiveIn(root, claimRel(req.Record), data, fileMode)"
  evidence: internal/core/implement/claim.go:210 — "case !now.Before(held.ExpiresAt):"
  evidence: internal/core/implement/claim_test.go:187 — "func TestAHeldClaimIsDeniedNamingTheHolder"
  evidence: internal/core/implement/claim_test.go:224 — "func TestALapsedLeaseIsClaimableAndTheLapseIsLogged"
  evidence: internal/core/implement/claim_test.go:140 — "func TestConcurrentClaimsOfOneRecordGrantExactlyOne"
- ac-4 — MET_WITH_CONCERNS: Check refuses the second session's release step always and a lane whose declared paths reach the reading corpus (derived from the presets file, failing closed when it cannot be read); the concern is that the corpus bound asks no question when the lane declares no paths, so the refusal rests on the second session declaring them, and the role itself is cooperative, not authenticated
  evidence: internal/core/implement/bounds.go:169 — "case StepRelease:"
  evidence: internal/core/implement/bounds.go:96 — "if len(paths) == 0 {"
  evidence: internal/core/implement/bounds.go:108 — "if hit := TouchesReadingCorpus(corpus, paths); hit != "" {"
  evidence: internal/core/implement/session.go:24 — "What the record gives is consistency, not authentication"
  evidence: internal/core/implement/session_test.go:114 — "func TestCheckHoldsTheSecondSessionsBounds"
  evidence: internal/core/implement/claim_test.go:302 — "func TestTheSecondSessionIsRefusedAReadingCorpusLane"
- ac-5 — MET_WITH_CONCERNS: the one-lane cap is enforced at claim time and test-held; the ceiling is recorded on join and reported by every check but, as the code states, never enforced because abcd runs no agent; a stop is Leave, which releases only that session's own claims and record, and the 'stops only itself' rule is otherwise a documented discipline in the command page rather than code
  evidence: internal/core/implement/claim.go:258 — "if c.Claim.Session == req.Session && c.Claim.Record != req.Record && now.Before(c.Claim.ExpiresAt) {"
  evidence: internal/core/implement/claim_test.go:256 — "func TestTheSecondSessionHoldsAtMostOneLane"
  evidence: internal/core/implement/session.go:94 — "so the ceiling is recorded and reported, never enforced"
  evidence: internal/core/implement/session.go:214 — "if c.Claim.Session != id {"
  evidence: commands/implement.md:111 — "a stop condition the second session meets stops only itself"
- ac-6 — MET_WITH_CONCERNS: a refused claim writes a backoff line for the second session naming the holder as the reason, but with minutes fixed at 0; a locked run state returns ErrContention with no log line at all, and queue contention is logged only if the session hand-writes `implement log backoff`, so 'contention of any kind' is logged by the verb for one kind and the minutes are never measured by it
  evidence: internal/core/implement/claim.go:230 — "r.append(req.Session, EventBackoff, map[string]any{"
  evidence: internal/core/implement/claim.go:232 — ""reason": "record claimed by session " + held.Session, "minutes": 0,"
  evidence: internal/core/implement/run.go:182 — "if errors.Is(err, fsutil.ErrLockContention) {"
  evidence: internal/core/implement/session_test.go:171 — "r.Log("beta", EventBackoff, map[string]string{"on": "queue""
- ac-7 — MET_WITH_CONCERNS: Compare derives per mode the windows, wall minutes, lanes landed, collisions, backoff and agent minutes from the log and names a Leader by lanes landed per hour, test-held on a fixture covering all four modes; the concern is that Leader is declared a figure, not a verdict, so 'names which it would keep' is left to the hand-written run report
  evidence: internal/core/implement/report.go:54 — "type ModeTally struct {"
  evidence: internal/core/implement/report.go:91 — "Leader is the mode with the most lanes landed per wall-clock hour"
  evidence: internal/core/implement/report.go:92 — "It is a figure, not a verdict: the run's report names which mode it would keep"
  evidence: internal/core/implement/report_test.go:16 — "func TestCompareDerivesEachModeFromAFixtureLog"
  evidence: internal/surface/cli/implement_surface_test.go:253 — "func TestImplementReportReadsANamedLog"

Gap audit:
- honoured:
  - the claim is an atomic exclusive create and a lease; a dead session strands nothing
    evidence: internal/core/implement/claim.go:17 — "taken by an exclusive create that fails if the file exists"
    evidence: internal/core/implement/claim_test.go:224 — "func TestALapsedLeaseIsClaimableAndTheLapseIsLogged"
  - a second session joins without the first being told
    evidence: internal/core/implement/session_test.go:15 — "func TestJoiningNeedsNoWordFromTheFirstSession"
  - the log records which mode a window ran, set by the first session only
    evidence: internal/core/implement/session.go:330 — "if s.Role != RoleFirst {"
  - only the first session may cut a release
    evidence: internal/core/implement/bounds.go:171 — ""only the first session cuts a release; leave the release step to it""
  - in a split-roles window the second session opens no lane
    evidence: internal/core/implement/claim_test.go:413 — "func TestTheSecondSessionOpensNoLaneInASplitRolesWindow"
  - the comparison is derived from the log, per mode
    evidence: internal/core/implement/report.go:9 — "The comparison is derived from the log and from nothing else"
  - the verbs are wired on the CLI and the plugin surface
    evidence: internal/surface/cli/implement.go:33 — "Use: "implement","
    evidence: commands/implement.md:110 — "An allowed step writes nothing."
- diverged:
  - the second session obeys its own ceiling above the first's: delivered as a recorded and reported figure, never enforced
    evidence: internal/core/implement/session.go:89 — "MaxCeiling bounds a stated agent ceiling"
    evidence: internal/core/implement/session.go:94 — "recorded and reported, never enforced"
  - a lane that touches the reading corpus is refused: delivered only for paths the second session declares; a claim or check with no paths asks no corpus question
    evidence: internal/core/implement/bounds.go:94 — "A lane that declares no paths asks no corpus question."
  - the backoff log names the minutes spent: the verb's own backoff line always carries minutes 0
    evidence: internal/core/implement/claim.go:232 — ""minutes": 0,"
  - the report names which mode it would keep: delivered as a per-hour leader figure, with the keep verdict left to the hand-written report
    evidence: internal/core/implement/report.go:92 — "It is a figure, not a verdict"
- missing:
  - contention of any kind backs off with a logged reason: a locked run state returns ErrContention and writes no backoff line
    evidence: internal/core/implement/run.go:182 — "if errors.Is(err, fsutil.ErrLockContention) {"
    evidence: internal/core/implement/run.go:183 — "return fmt.Errorf("%w: the run state is locked by another session's change; back off and retry", ErrContention)"
  - at the window's end the log carries the lanes each session took: no window-end event exists; the per-window lanes are derived by Compare from hand-logged lane_open lines
    evidence: internal/core/implement/log.go:41 — "EventLaneOpen = "lane_open""
    evidence: internal/core/implement/report.go:149 — "closeWindow := func() {"

Scope-condition dispositions:
- cond-2609221657584605 — survived: the run state lives under the user's machine store and the role is a cooperative record two sessions of one account keep, exactly the one-account one-machine shape the condition assumed
  evidence: internal/core/implement/run.go:40 — "const runsRelPath = ".abcd/runs""
  evidence: internal/core/implement/session.go:25 — "is consistency, not authentication — two sessions of one account can each"
- cond-2609221657584877 — survived: the release step is refused to the second role and open to the first, so the first session remains the one that may cut a release
  evidence: internal/core/implement/bounds.go:169 — "case StepRelease:"
  evidence: internal/core/implement/session_test.go:137 — "e.String("condition") != "second_session_release""
- cond-2609221657583516 — survived: the log is one file per UTC day, one JSON object per line, appended in a single O_APPEND write, and a two-writer test shows lines never interleave
  evidence: internal/core/implement/log.go:19 — "The run log is one file per UTC day, one JSON object per line, append-only."
  evidence: internal/core/implement/log.go:208 — "func logFileName(ts time.Time) string { return ts.UTC().Format(time.DateOnly) + ".jsonl" }"
  evidence: internal/fsutil/appendline_test.go:93 — "func TestAppendLineInTwoWritersNeverInterleave"
## Grounds

- pursued: the pilot's ceiling wait was a third of its clock and a second session is the cheapest test of whether more hands help before the verb owns pacing; we expect throughput without collisions; shown wrong if collisions and wasted agent minutes exceed the lanes the second session lands
