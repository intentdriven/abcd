---
id: itd-2611041549461012
slug: when-the-session-coordinating-unattended
spec_id: spc-2611041706004822
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201925079472, itd-2611031342011261, itd-2610180912189125]
refines: [itd-2610222215485265]
related_issues: [iss-2611022247128872, iss-2611032348373007, iss-2610110915210571]
severity: major
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# A session that takes over a run steps on from what abcd wrote down

## Press Release

> **When the session coordinating unattended runs dies, the next session picks the runs up from what abcd wrote down, and nothing is started twice.** Alice runs abcd unattended on a second account of their computer. When the session driving the runs is gone (a crash, a lost connection, a compaction that forgot), the next session types `abcd implement resume`: it lists every run this machine drives, each lane's stage, each agent still alive or lost, the receipts that arrived while nobody collected them, the pull requests waiting on the queue and how far each lane's base has moved, and the work each lane left uncommitted, and it steps on from there. An agent whose receipt is waiting is collected before any fresh one starts, and work left uncommitted is saved aside and named before anyone builds over it. "I used to find a second implementer paid for the work the first had finished, and a step loop nobody owned still launching agents," said Alice, a solo founder. "Now a restart reads the record instead of guessing."

## Why This Matters

The restart half of an unattended run has its evidence in the loop's own issues. A step loop a stopped session left in the background kept stepping its run and launching agents on a key nobody watched, and the successor could not learn from the run who drove it, from which process, or which key its launches spent (iss-2611022247128872); the run state's `Driver` is a string naming the kind of driver, not a live record. A finished agent was stranded when the step that would collect it was refused, and every later step asked for a fresh implementer though the receipt sat at the awaited path, which an unattended run would have paid for twice (iss-2611031337115764). Stopping a stage left no record of what it wrote, and a resume would have started a fresh implementer over ten changed files (iss-2611032348373007). A session's work in flight does not survive the host's compaction unless it wrote it down (iss-2610222026223032). A crashed copy stranded reserved records and left pull requests drifting (iss-2610241107435264, iss-2610241414414823), and a resume says nothing of how far a lane's base has moved (iss-2611040337228910). On 2026-11-03 the coordinator itself was a host session plus a polling script, so there is no coordinator to restart, only a session to replace; iss-2611031146255825 names the missing drive.

What already exists is the pieces: the run state and the drain state, each lane's checklist (`abcd checklist`), the restart of a gone implementer with its uncommitted work saved aside (`internal/core/implement/loop/restart.go`), the drain's first-signal cancel and second-signal kill, the run log's `stop` and `intervention` events (`abcd implement log`), and the launch ledger (itd-2611031342011261), whose entries carry each agent's process id so a reader judges liveness itself. The drain spec (spc-2610230813348813) already promises that a signal leaves a lane resumable and a second drive resumes it. This intent makes those pieces one path; its sibling, the stop switch (itd-2611041549471712), is the person's way to end the work on purpose. The drive's self-update (itd-2611062149308723) builds on it: a drive that re-executes itself on a new release resumes from the same recorded state this path reads.

## Mechanism

We expect a successor to step on without starting anything twice because every fact it needs is a record the run wrote as it went, the live driver on the run state, each launch and its process in the ledger, each lane's work on its checklist and in the receipt path the await names, and the resume reads those records in one listing before it moves, collecting what arrived before it launches. The claim is shown wrong if a resume starts a fresh agent for an await whose receipt is already at its path, or builds a stage over uncommitted work without naming it first.

## Scope Conditions

- Holds for runs abcd drives on one machine: build runs, drain lanes and the terminal drive (itd-2610222215485265), whose plugin page stays an agent's command and whose terminal form stays the person's way to drive it (dcn-2611041550139264). A hosted runner's run is resumed by its host. <!-- cond: cond-2611041706003922 -->
- A process killed outright is outside the promise; the resume treats its agents as lost and its work as saved aside. <!-- cond: cond-2611041706002354 -->
- Recorded state means abcd's own files: the run and drain states, the checklist, the launch ledger and the log; never a session's transcript or memory. <!-- cond: cond-2611041706009161 -->
- Holds on macOS and Linux, where `~/.abcd.noindex/` is abcd's home; there is no Windows target. <!-- cond: cond-2611041706005578 -->

## Acceptance Criteria

- **Given** a run, **when** a session drives it, **then** the state records the live driver (session, process id, start time) and each in-flight launch with its route, and a second driver is refused while the first is alive (iss-2611022247128872).
- **Given** a driver that is gone, **when** the next session runs the resume, **then** it lists every run on the machine with each lane's stage, each agent alive or lost by its process, each receipt that arrived uncollected, each pull request waiting and how far each lane's base has moved (iss-2611040337228910), and the uncommitted work each lane holds; it collects an arrived receipt before it ever starts a fresh agent (iss-2611031337115764).
- **Given** a lane with uncommitted work and no live agent, **when** the resume steps it, **then** the work is saved aside and named in the record before a fresh agent starts, never built on silently (iss-2611032348373007).
- **Given** a drain stopped while awaiting a judgement, **when** it is resumed, **then** the judgement can be given or the entry is cleared (iss-2610110915210571).
- **Given** a coordinator session that lost its context, **when** it resumes, **then** everything it needs is in the listing above and in the checklist, so nothing depends on the transcript (iss-2610222026223032).
- **Given** a run the stop switch halted (itd-2611041549471712), **when** the stop is lifted and a session resumes, **then** the listing shows what the stop left in each lane and the reservations it released, and the resume reserves again before it steps.

## Open Questions

_None open. Every question the scratch draft of 2026-11-04 put for this half had one answer on the record: a stop takes hold at a tool-call boundary, never within one (dcn-2611031722512037); a coordinating session is walled where abcd starts it, and one the person starts off a profile is warned (dcn-2611031722512037, dcn-2611041550129278); the run's account of its launches is the ledger (itd-2611031342011261, decisions 7 to 12); a gone agent restarts fresh from its last commit with its uncommitted work saved aside (iss-2610080620372731)._

## Decisions

- 2026-11-04, the product thinker (dcn-2611041550139264): the restart path and the stop switch are two records, this one and itd-2611041549471712, as the scratch draft's decomposition proposed; the routing is confirmed.
- 2026-11-04, the product thinker (dcn-2611041550139264): a stop releases the run's claims on its work, so a resume after a stop reserves again before it steps.
- 2026-11-04, the product thinker (dcn-2611041550139264): the drain's plugin page stays an agent's command and the terminal form stays; the runs this path resumes include the terminal drive.

## Decomposition

Hand-run four-piece table, 2026-11-04, from the scratch draft that held the stop switch and the restart path together; the product thinker confirmed the split (dcn-2611041550139264).

| Part | Kind | Home |
|---|---|---|
| the resume listing, the live-driver record, collect-before-launch, saved-aside work, the drain's judging entry | capability | this intent |
| one command stops the runs of each account that chose the folder at their next move, lifted by the person's own command | capability | itd-2611041549471712, the stop switch |
| a stop is a recorded state read at each move, never a signal to a terminal; lifting it is the person's, by rule | trust rule | an ADR owed with the stop switch, extending adr-2610240850193281 (the grant permits, this withdraws) and dcn-2611031722512037; a brief invariant when accepted; not filed here |
| a run's account of itself lives in abcd's files, never in a session's memory | stance | a candidate principle, for the product thinker to confirm; not filed |
| the driver fields, the resume's output | plumbing | the brief's loop chapter, through this intent's spec |

Verdict: SPLIT, confirmed. Typed links: `refines` itd-2610222215485265 (its spec's resume story becomes this path for every run); `builds_on` itd-2609201925079472 (the loop's pacing and checkpoints), itd-2611031342011261 (the ledger that names each agent's process) and itd-2610180912189125 (the autonomous mode's store, the shape a recorded state mirrors); `related_issues` as the frontmatter lists, whose remedies are the criteria above. Advisory reversal flags: none.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-fffd97958311 -->
Fidelity review — receipt rcp-fffd97958311 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:30bfb9f4ab659690e2ff03bba10dc2d6b38b10b08705fd906fd9d1df0e1df035
Input attestations: diff:lane-1 e02e5c8c8cc29733aa56a60442407a54d9a21fce..99dbd32c6dd62348926326c50bd4d7560a877470 (spc-2611050505250705 step 1, PR #1451)@-; diff:lane-2 5c2aaed49f11b64077f1f57742b3fce6de483f7a..9e665a6967db69b37af740fedb7355a55bb1d120 (spc-2611050505250705 step 2); the brief's wider range e4da1db7289294b5bbf5da15c37e7b95b3e15f88..9e665a69 also carries lane-1 and main's PR #1450 (.githooks, .abcd/rules.json, buildcache_test.go, two dcn records), none of it this intent's delivery; no binary file in either range@-; commit:f175c4db5c64edfd20fd1f509472af560a6b8a1c and 205bd50bd12fbd5df6a2fa80fa9f26e5d2c10996 (spc-2611041706004822 steps 1-3, closed spec, read at tip 9e665a69)@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: control.json records the live driver (session, pid, start time), and each runner launch with its runner and route; ClaimDriver refuses a second live driver, and both the step and the terminal drive call it, under a test. Concerns: a launch is recorded only for an agent started through a runner, so host-delegated agents have no launch entry; Route holds only an override, so it is empty on a default route; and iss-2611022247128872, which this criterion cites, is still in open/ even though its remedy also asks for implement status to show these fields.
  evidence: internal/core/implement/loop/control.go:48 — "type LiveDriver struct { Session ... PID int ... StartUnix int64"
  evidence: internal/core/implement/loop/control.go:65 — "type Launch struct { Lane, Role, Runner, Route, Session, PID, StartUnix"
  evidence: internal/core/implement/loop/control.go:288 — "back off: only one driver steps a run; once that driver is gone, `abcd implement resume` lists what it left"
  evidence: internal/surface/cli/build.go:1257 — "if err := loop.ClaimDriver(root, id, loop.Options{}); err != nil {"
  evidence: internal/core/implement/loop/terminal.go:921 — "if err := ClaimDriver(t.repoRoot, runID, t.o.Options); err != nil {"
  evidence: internal/core/implement/loop/runner_pause.go:437 — "recordLaunch(root, st, Launch{Lane: laneID, Role: aw.Role, Runner: route.Runner, Route: route.Override(), Session: session, At: now})"
  evidence: internal/core/implement/loop/resume_test.go:53 — "func TestAClaimRecordsTheDriverAndRefusesASecond(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2611022247128872-a-step-loop-a-stopped-session-left-in.md:13 — "remedy: "Record the run's live driver ... show them in implement status, and refuse a second driver while the first is alive.""
- ac-2 — MET_WITH_CONCERNS: ListRuns walks every checkout this machine noted and lists each lane's stage, each agent's state, its receipt_arrived flag, its PR, base_moved and its uncommitted count. Lane-2 adds the ledger judgement that rates a runner's agent by its own process, and the take-over collects arrived receipts before any restart, under tests. Concerns: the bare listing names a PR only by the number in the state; whether it is waiting on the queue is read from the forge only by --take (dcn-2611071009208361). base_moved counts against origin as last fetched, with no fetch. A host-delegated agent still has no process of its own to be judged by, so it reads 'unknown' rather than alive or lost. iss-2611040337228910 stays open, since its sync offer is not built.
  evidence: internal/core/implement/loop/resume.go:247 — "for _, checkout := range machineCheckouts(repoRoot, home) {"
  evidence: internal/core/implement/loop/resume.go:71 — "BaseMoved *int `json:"base_moved,omitempty"`"
  evidence: internal/core/implement/loop/resume.go:74 — "Uncommitted *int `json:"uncommitted,omitempty"`"
  evidence: internal/core/implement/loop/resume.go:386 — "gitutil.Run(checkout, "rev-list", "--count", lane.BaseSHA+".."+view.commit, "--")"
  evidence: internal/core/implement/loop/resume.go:814 — "func (p *agentProcs) judge(checkout, runID, laneID string, a Await, c control, tab *sessionname.Table) (AgentState, *runner.LedgerEntry, bool) {"
  evidence: internal/core/implement/loop/resume.go:592 — "res.Collected = collectArrived(repoRoot, runID, DefaultStages(), seen, procs, o)"
  evidence: internal/core/implement/loop/resume.go:80 — "Forge *LaneForge `json:"forge,omitempty"`"
  evidence: internal/core/implement/loop/resume_test.go:430 — "func TestTheListingSaysHowFarALanesBaseHasMoved(t *testing.T) {"
  evidence: internal/core/implement/loop/resume_test.go:201 — "func TestATakeOverCollectsBeforeItRestarts(t *testing.T) {"
  evidence: internal/core/implement/loop/resume_test.go:1050 — "func TestTheListingJudgesARunnersAgentByItsOwnProcess(t *testing.T) {"
  evidence: internal/core/implement/loop/resume_test.go:1137 — "func TestATakeOverCollectsWhatTheLedgerSaysHasEnded(t *testing.T) {"
- ac-3 — MET: The take-over saves a lost agent's uncommitted work aside through restartLane before re-telling its await. It also saves aside the work of a lane that has no agent, and writes a run-record entry naming the path. Work it could not save is a refusal that names no step. Tests cover each path, and lane-2's 'gone' case covers a runner's agent whose harness has died.
  evidence: internal/core/implement/loop/resume.go:647 — "aside, err := saveAside(repoRoot, root, runID, lane.ID, Await{}, p, "taken over: "+WhyDied, now)"
  evidence: internal/core/implement/loop/resume.go:687 — "r, err := restartLane(repoRoot, runID, laneID, "", o)"
  evidence: internal/core/implement/loop/resume_test.go:232 — "func TestATakeOverSavesAsideBeforeAFreshAgent(t *testing.T) {"
  evidence: internal/core/implement/loop/resume_test.go:261 — "func TestATakeOverRefusesOverWorkItCouldNotSave(t *testing.T) {"
  evidence: internal/core/implement/loop/resume_test.go:1098 — "func TestATakeOverRestartsAnAgentWhoseOwnProcessIsGone(t *testing.T) {"
- ac-4 — MET: A drain that ends stopped clears Judging, and so does the stop path. The listing names a drain that is awaiting a judgement, and gives the --judgement move that answers it. iss-2610110915210571 is resolved, naming d7c05e9cb (which predates this delivery, as spc-2611050505250705 records).
  evidence: internal/core/implement/loop/drain.go:515 — "if st.Stopped != "" && st.Judging != nil { st.Judging, write = nil, true"
  evidence: internal/core/implement/loop/stop.go:1078 — "if st.Judging != nil { st.Judging = nil"
  evidence: internal/core/implement/loop/resume.go:296 — "l.Next = "give the judgement over " + l.Judging + " with `abcd drain --judgement`, in that checkout""
  evidence: internal/core/implement/loop/resume_test.go:466 — "func TestTheListingNamesAJudgement(t *testing.T) {"
  evidence: .abcd/work/issues/resolved/iss-2610110915210571-a-drain-that-an-outage-stops-while-it.md:18 — "commit: "d7c05e9cb758ed0fe469a7a32c6d2c24cc0ffd06""
- ac-5 — MET_WITH_CONCERNS: The listing is built only from abcd's files (state, control.json, the ledger, the drain state), and each run carries a Next move, so the resume path reads no transcript. Concerns: nothing in the delivery tests a coordinator that lost its context end to end. The listing does not point to or read the checklist that the criterion pairs it with. iss-2610222026223032, which the criterion cites, is still open, and its hook-and-to-do-list remedy is not part of this delivery.
  evidence: internal/core/implement/loop/resume.go:208 — "machine with anything left. The record is the whole of it, so a session that lost its context needs nothing else."
  evidence: internal/core/implement/loop/resume_test.go:165 — "func TestTheListingIsWhatARunWroteDown(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2610222026223032-autonomous-work-does-not-reliably.md:13 — "remedy: "each abcd session keeps its to-do list as a file in the local tier"
- ac-6 — MET: The listing carries the run's Stop: HaltedAt, Released, and Left per lane with its aside, plus the Reserved flag. After a stop has halted the run, the take-over reserves again before it saves aside or restarts any lane, and backs off at exit 3 when a peer holds the work. A test lifts a stop, reads the listing's account of it, takes the run over and finds origin's reservation taken again.
  evidence: internal/core/implement/loop/control.go:83 — "type RunStop struct { ... Released string ... Left []StopLeft"
  evidence: internal/core/implement/loop/resume.go:598 — "switch err := reserveTaken(repoRoot, &start, o, false); {"
  evidence: internal/core/implement/loop/resume_test.go:332 — "func TestATakeOverAfterAStopReservesAgainFirst(t *testing.T) {"
  evidence: internal/core/implement/loop/resume_test.go:376 — "func TestATakeOverBacksOffWhenAPeerTookTheWork(t *testing.T) {"

Gap audit:
- honoured:
  - One live driver is recorded per run, and a second is refused while the first is alive
    evidence: internal/core/implement/loop/control.go:288 — "back off: only one driver steps a run"
  - An agent whose receipt is waiting is collected before any fresh one starts
    evidence: internal/core/implement/loop/resume.go:592 — "res.Collected = collectArrived(repoRoot, runID, DefaultStages(), seen, procs, o)"
  - Uncommitted work is saved aside and named before anyone builds over it
    evidence: internal/core/implement/loop/resume.go:647 — "aside, err := saveAside("
  - The launch ledger names each agent's process, so a reader judges liveness itself
    evidence: internal/core/implement/loop/resume.go:814 — "func (p *agentProcs) judge("
  - A resume after a stop reserves again before it steps
    evidence: internal/core/implement/loop/resume.go:598 — "reserveTaken(repoRoot, &start, o, false)"
- diverged:
  - The resume lists the pull requests waiting on the queue: the bare listing names only the PR number from the state, and only the take-over reads queue state from the forge
    evidence: .abcd/work/decisions/dcn-2611071009208361-the-take-over-not-the-bare-listing-reads.md:6 — "The take-over, not the bare listing, reads the forge for each lane's pull request"
    evidence: internal/core/implement/loop/resume.go:706 — "readForge(repoRoot, now, res.Run, &res.Left)"
  - How far each lane's base has moved: measured against origin's default branch as last fetched, with no fetch
    evidence: internal/core/implement/loop/resume.go:386 — ""rev-list", "--count", lane.BaseSHA+".."+view.commit"
  - Each agent is judged alive or lost by its own process: only agents started through a runner have a ledger entry; the stop still judges by the run's record, and host agents can read 'unknown'
    evidence: internal/core/implement/loop/stop.go:415 — "collected = collectArrived(repoRoot, runID, steps, seen, nil, o)"
    evidence: internal/core/implement/loop/stop.go:487 — "AgentUnknown AgentState = "unknown""
- missing:
  - Records a crashed copy stranded are swept (Why This Matters cites iss-2610241107435264); left out and still open
    evidence: .abcd/work/decisions/dcn-2611071009208361-the-take-over-not-the-bare-listing-reads.md:6 — "Sweeping records a crashed copy stranded (iss-2610241107435264) is left out of this step"
  - Drifting pull requests are synced with the default branch (iss-2610241414414823); they are only named, and the issue stays open
    evidence: .abcd/work/decisions/dcn-2611071009208361-the-take-over-not-the-bare-listing-reads.md:6 — "iss-2610241414414823 also stays open: its remedy asks for a sweep that merges the default branch into each drifting pull request"
  - The resume's text form shows the agent's own process and launch; only --json carries them
    evidence: internal/core/implement/loop/resume.go:58 — "Launch string `json:"launch,omitempty"`"

Scope-condition dispositions:
- cond-2611041706003922 — survived: The listing reads this machine's checkouts and drain states, and the step and the terminal drive both claim the driver; nothing in the delivery reaches a hosted runner's run.
  evidence: internal/core/implement/loop/stop.go:998 — "func machineCheckouts(repoRoot, home string) []string {"
  evidence: internal/core/implement/loop/terminal.go:921 — "if err := ClaimDriver(t.repoRoot, runID, t.o.Options); err != nil {"
- cond-2611041706002354 — narrowed: A killed agent is treated as lost and its work saved aside only where something on record says it is gone. An agent with no process of its own on record, inside its claim's term or its lease, or under a driver no session names, is 'unknown': it is left 'may still be in its call' and not restarted.
  narrowing: holds for runner agents whose ledger entry names their own process, for a host agent whose session-named driver is gone, and otherwise only once the claim's term or the reservation lease has run out; before that the agent is left as unknown, not treated as lost
  evidence: internal/core/implement/loop/resume.go:833 — "return AgentLost, &e, true"
  evidence: internal/core/implement/loop/resume.go:670 — "may still be in its call, so it is not restarted"
  evidence: internal/core/implement/loop/stop.go:487 — "AgentUnknown AgentState = "unknown""
- cond-2611041706009161 — survived: The listing and the take-over read only the run state, control.json, the launch ledger under abcd's home and the drain state; lane-2 adds the ledger, which is one of the named files, and TestMain keeps tests off the machine's own ledger.
  evidence: internal/core/implement/loop/resume.go:246 — "procs := ledgerOf(o, runner.LedgerQuery{All: true})"
  evidence: internal/core/implement/loop/drive_test.go:72 — "launchLedgerHome = func() string { return "" }"
- cond-2611041706005578 — untested: Nothing in the delivered diffs exercises or contradicts the macOS/Linux-only assumption; the ledger home comes from layered.HomeRoots, and no platform-specific path was added.
<!-- abcd-review-end receipt=rcp-fffd97958311 -->
