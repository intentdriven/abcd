---
id: itd-2611041549471712
slug: from-their-own-account-a-person-types
spec_id: spc-2611041705580524
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201925079472, itd-2611031342011261, itd-2610180912189125]
refines: [itd-2610222215485265]
reverses: [dcn-2610222224054666]
severity: major
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# A person stops the unattended runs of each account that chose a folder, with one command

## Press Release

> **A person stops the unattended runs of each account that chose a shared folder, from their own account with one command, and it stays until their own command lifts it.** Alice runs abcd unattended on a second account of their computer, and at setup they chose a shared folder on it (`/Users/Shared` on macOS, its counterpart on Linux) for the things the two accounts pass between them. From their own account they type `abcd implement stop`: each of those runs finishes the tool call in hand, opens no new lane, starts no new agent, writes what each lane holds uncommitted into its record, releases what it reserved, and says so in the next daily digest. Lifting it is the person's act, and the guard's rule refuses it to an agent's session, a driver or a hook. "I could never reach a run from the terminal it was not in," said Alice, a solo founder. "Now a stop is a file my account writes and each of those runs reads before its next move, and it stays stopped until I say otherwise."

## Why This Matters

The product thinker ruled that watching is no control: a person cannot interrupt a tool call as it runs, so a person watching never makes an unwalled agent acceptable (dcn-2611031722512037). The same fact cuts the other way: a stop can only take hold at a tool-call boundary, so it has to be a recorded state every run reads before each move, not a key the person presses in a terminal they are not at. Today the one such state is the autonomous mode (`abcd mode autonomous`, itd-2610180912189125, adr-2610181003541997), which grants and queues but stops nothing, and its store admits only files this account owns that no one else can write. The person's seat and the run's seat are two accounts on one machine, so no store abcd has today can carry a stop from one to the other; iss-2610121528179959 names the same gap for handovers. A step loop a stopped session left behind kept launching agents on a key nobody watched (iss-2611022247128872).

The product thinker settled the place at the filing (dcn-2611041550139264): the stop lives on the same machine, in a shared folder chosen as an option at setup, on a machine no other person shares, and the store ownership rule admits this one exception. A stop that reaches the machine over a private network or through a cloud adapter is later work. The daily digest (itd-2611041549483696) lives in the same folder, so one choice holds for both.

## Mechanism

We expect a stop to take hold on the runs that read where it is written because each run already reads a recorded state before each move (the autonomous mode's store is the shape), and a file in the shared folder written by the person's account is one more such read; the runs stop at their next boundary, which is the only place a stop can act. The claim is shown wrong if a run that reads the stop file makes a move after it exists, or if abcd removes it by any path but the person's own lift.

## Scope Conditions

- Holds for runs abcd drives on one machine: build runs, drain lanes and the terminal drive (itd-2610222215485265); a hosted runner's run is stopped by its host. <!-- cond: cond-2611041705583688 -->
- Holds where the person chose the shared folder at setup; a machine without one has no stop that crosses accounts, and `abcd ahoy` says so. The folder is one option of setup, never a requirement. <!-- cond: cond-2611041705589732 -->
- Holds on a machine no other person shares: the folder is readable by every account on it, and the ruling that admits it rests on the machine being the person's alone. <!-- cond: cond-2611041705586060 -->
- A stop takes hold at the next move of each run; an agent mid-call finishes that call. A process killed outright is outside the promise, and the restart path (itd-2611041549461012) treats its agents as lost. <!-- cond: cond-2611041705583744 -->
- The stop is the person's act from their own account; an agent's session is refused the command. Lifting it is the same person's act. <!-- cond: cond-2611041705584558 -->
- Holds on macOS and Linux; there is no Windows target. <!-- cond: cond-2611041705581125 -->

## Acceptance Criteria

- **Given** the person runs the stop from their own account, **when** a run of their account, or of an account that chose the same shared folder, takes its next move, **then** it opens no lane, hands out no stage, starts no agent, and writes a `stop` event naming who set it and when; an agent already running finishes its call and its receipt is collected, never acted on further; and only then, with no agent live in the lane, is each lane's uncommitted work saved aside as a restart does and its reservations released so a peer may take the work (dcn-2611041550139264).
- **Given** the stop holds, **when** an agent's session, a driver or a hook tries to lift it, **then** the guard's rule refuses it; the person's own command lifts it, and the lift is a recorded event.
- **Given** a terminal run, **when** Ctrl-C or a hang-up reaches it, **then** the signal is a stop: the running agent's call finishes and its receipt is collected, then the lane's uncommitted work is saved aside and its claims released, and the drive prints the run and the lane; a second press kills the process group at once (dcn-2611041608057544). This reverses dcn-2610222224054666, under which the open lane kept its reservation for the next drive to resume.
- **Given** the stop holds, **when** the last run in the account that runs the work ends under it, **then** the stop composes that day's digest page before it ends, in that account (itd-2611041549483696), naming the stop, every run it halted and what each left; a page already written for the day is rewritten whole.
- **Given** setup, **when** the person is offered the shared folder, **then** the walkthrough says what it is for, that it is read by every account on the machine, and that it is meant for a machine no other person shares; declining leaves every other part of abcd working, and the stop then reaches only runs in the person's own account.
- **Given** the stop file's home, **when** abcd reads it, **then** it accepts this one file outside its owner-only rule, written by the person's account, and refuses a stop file any other account wrote, naming it.

## Open Questions

_None open. The three questions the scratch draft of 2026-11-04 put are the first three decisions below._

## Decisions

- 2026-11-04, the product thinker (dcn-2611041550139264): the stop lives on the same machine, in a shared folder chosen as an option at setup (`/Users/Shared` on macOS, its counterpart on Linux), on a machine no other person shares; the store ownership rule admits this one exception; a stop over a private network or through a cloud adapter is later work. The scratch draft's second option, made deliberate.
- 2026-11-04, the product thinker (dcn-2611041550139264): a stop lets an agent in a call finish that call, then stops.
- 2026-11-04, the product thinker (dcn-2611041550139264): a stop releases the run's claims on its work.
- 2026-11-04, the product thinker (dcn-2611041550139264): the stop switch and the restart path are two records; the routing is confirmed.
- 2026-11-04, the product thinker (dcn-2611041608057544): Ctrl-C in a terminal run is a stop: the call finishes and claims are released; a second press kills at once.

## Decomposition

Hand-run four-piece table, 2026-11-04, from the scratch draft that held the stop switch and the restart path together; the product thinker confirmed the split (dcn-2611041550139264).

| Part | Kind | Home |
|---|---|---|
| one command stops the runs of each account that chose the folder at their next move, lifted by the person's own command; the shared folder offered at setup | capability | this intent |
| the resume listing, the live-driver record, collect-before-launch, saved-aside work | capability | itd-2611041549461012, the restart path |
| a stop is a recorded state read at each move, never a signal to a terminal; lifting it is the person's, by rule; one shared folder on an unshared machine is the one file outside the owner-only rule | trust rule | an ADR owed, extending adr-2610240850193281 (the grant permits, this withdraws) and dcn-2611031722512037, and narrowing the store ownership rule by this one exception (dcn-2611041550139264); a brief invariant when accepted; not filed here |
| a run's account of itself lives in abcd's files, never in a session's memory | stance | a candidate principle, for the product thinker to confirm; not filed |
| the stop file's shape, the setup walkthrough's wording, the digest's line | plumbing | the brief's loop chapter, through this intent's spec |

Verdict: SPLIT, confirmed. Typed links: `refines` itd-2610222215485265 (a stop ends its drive the same way); `builds_on` itd-2609201925079472, itd-2611031342011261 and itd-2610180912189125; `related_issues` iss-2611022247128872 and iss-2610121528179959. `reverses` dcn-2610222224054666 (dcn-2611041608057544): a signal no longer leaves the lane's reservation held; the call finishes and the claims are released. Flagged, never auto-classified: this intent narrows the autonomous mode's reading that only `abcd mode back` ends it; a stop ends the work without ending the mode.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4a92f9361178 -->
Fidelity review — receipt rcp-4a92f9361178 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:d126a4f4a45d2697d6cddf186ff1d208a8438aa64b02aa1aabc3347631c0bba9
Input attestations: diff:6259a0391..255efd43556d0a2762039a398fa6670eef641a40@-; spec:spc-2611041705580524 (closed earlier, on the default branch)@-; spec:spc-2611050505220309@-;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: stopCheck guards the step, the agent's start, the review's send, a restart, a run's start and a drain's move; haltRun records who and when, collects receipts, waits on a live agent, saves aside and releases last; focused tests pass on the copy. Concerns: a run a drive stopped on, a changed configuration and a failed save each leave the work unsaved or the claim held by design, the first of them lapsing with its lease; the shared-log line is written only where the driver has a session, so a plain terminal drive has the run record alone; and the crossing of accounts is proved only by a stand-in owner handed to the judgement, never by a real second account.
  evidence: internal/core/implement/loop/stop.go:111 — "func stopCheck(repoRoot, runID string, steps Stages, o Options) (string, error) {"
  evidence: internal/core/implement/loop/stop.go:366 — "a stop set by %s at %s: no lane opens, no stage is handed out and no agent starts"
  evidence: internal/core/implement/loop/stop.go:606 — "if live {"
  evidence: internal/core/implement/loop/stop.go:620 — "out.Settled = saveLaneForStop(repoRoot, root, st, i, why, &out, now)"
  evidence: internal/core/implement/loop/stop.go:635 — "clause, failed := releaseReservation(repoRoot, st.Key, "", now)"
  evidence: internal/core/implement/loop/runner_pause.go:396 — "if _, err := stopCheck(repoRoot, runID, steps, o); err != nil {"
  evidence: internal/core/implement/loop/drain.go:442 — "stopNote, serr := drainStop(repoRoot, &st, o)"
  evidence: internal/core/implement/loop/loop.go:505 — "A stop opens no run, a resumed one included (stop.go)."
  evidence: internal/core/implement/loop/stop_test.go:110 — "func TestAStopHoldsAnAgentInItsCall(t *testing.T) {"
  evidence: internal/core/implement/loop/stop_test.go:234 — "func TestAStopReleasesTheReservationOnlyAtTheEnd(t *testing.T) {"
  evidence: internal/core/implement/loop/stop.go:297 — "func stopOverDriveStop(runID string, s runstop.Stop) error {"
  evidence: internal/core/implement/loop/stop.go:433 — "if driver == nil || driver.Session == "" {"
  evidence: .abcd/development/specs/closed/spc-2611041705580524-from-their-own-account-a-person-types.md:37 — "the tests stand a second account in through the owner the judgement is handed"
- ac-2 — MET: The registry and the any-program acts refuse the verb in each spelling, including through the source wrapper; a shell write and a file tool's write to the stop's files are refused; the verb itself refuses a caller with no terminal or under a host session and refuses a lift where it cannot tell; no loop code removes the file; a lift is logged before the file goes and fails if it cannot be logged. The head's guard blocked the verb and a mv of the stop file when asked, and the named tests pass.
  evidence: internal/core/guard/defaults/guard.json:1299 — ""abcd-implement-stop": {"
  evidence: internal/core/guard/personacts.go:85 — "verbs: []string{"implement", "stop"}, adjacent: true,"
  evidence: internal/core/guard/allowonce.go:217 — "func (r Registry) stopStoreSignal(command string, rc *readCtx, v verdicts) (payloadSignal, bool) {"
  evidence: internal/surface/cli/hook_occupancy.go:178 — "if guard.StopStorePath(path) || guard.StopStorePath(fsutil.RealExistingPath(path)) {"
  evidence: internal/surface/cli/implement_stop.go:201 — "if !c.person() {"
  evidence: internal/core/runstop/runstop.go:759 — "if err := logEvent(home, "lifted", now); err != nil {"
  evidence: internal/core/guard/personacts_test.go:150 — "func TestTheStopIsThePersonsAct(t *testing.T) {"
  evidence: internal/surface/cli/implement_stop_test.go:351 — "func TestGuardHookDeniesTheFileToolsTheStop(t *testing.T) {"
- ac-3 — MET_WITH_CONCERNS: The first interrupt, hang-up or termination closes the stop channel and leaves the context live so the call finishes; the drive's end takes haltOnSignal through haltRun in the stop's order and renderDriveEnd prints the run and the lane; the second signal cancels the context, kills the group and exits; the reservation is released, reversing the earlier ruling. Concern, found by reading, with no test of it found: when a step ends with the runner returning no receipt, its claim is dropped and the await stays, and agentState reads an await with no claim under a live driver as unknown; in a drive that driver is the drive's own process, so the first press settles nothing, keeps the claims and says an agent is still in its call when none is.
  evidence: internal/surface/cli/drain_drive_signal.go:51 — "signal.Notify(s.ch, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)"
  evidence: internal/core/implement/loop/terminal.go:447 — "func (t *terminal) haltOnSignal() {"
  evidence: internal/surface/cli/drain_drive.go:599 — "left on its branch: %s, run %s, %s"
  evidence: internal/surface/cli/drain_drive_signal_test.go:181 — "func TestSignalsThroughARealDriveAndARealProcessGroup(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_stop_test.go:72 — "func TestAStopLetsTheCallFinishAndReleasesTheLanesReservation(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_personstop_test.go:182 — "func TestASignalHaltsTheLaneAndTheNextDriveTakesItOver(t *testing.T) {"
  evidence: internal/core/implement/loop/runner_pause.go:594 — "the runner ran the "+aw.Role+" but no receipt was handed back"
  evidence: internal/core/implement/loop/terminal.go:879 — "if t.stopped() {"
  evidence: internal/core/implement/loop/stop.go:501 — "if tab == nil || tab.Alive(c.Driver.host()) {"
  evidence: internal/core/implement/loop/terminal.go:464 — "%s: an agent is still in its call, so the next drive settles what it left"
- ac-4 — MET_WITH_CONCERNS: Each halting rewrites the day's page whole from the records, so the last run to end leaves one naming the stop, every halted run and what each left, written atomically in the account running the work; tests hold it across midnight and a second stop. Concerns: where no shared folder was chosen nothing is written, though the criterion carries no such condition; the page is only the stop's account, since the daily digest it is part of is still a draft; a run in another checkout is found only through the list a driver's claim adds to, and a claim under a stop adds nothing, so a run that never had a driver there can drop off a later rewrite; and the directory is per account, not the single directory the spec named.
  evidence: internal/core/implement/loop/stop.go:826 — "func composeStopPage(repoRoot string, rec RunStop, o Options) error {"
  evidence: internal/core/implement/loop/stop.go:877 — "_, err := runstop.WritePage(home, now, []byte(page))"
  evidence: internal/core/runstop/runstop.go:825 — "if err := fsutil.WriteFileAtomicInRoot(root, rel, data, sharedFileMode); err != nil {"
  evidence: internal/core/implement/loop/stop_test.go:355 — "func TestTheStopComposesTheDaysPage(t *testing.T) {"
  evidence: internal/core/implement/loop/stop_test.go:383 — "func TestAStopOverMidnightNamesEveryRunItHalted(t *testing.T) {"
  evidence: internal/core/runstop/runstop_test.go:322 — "func TestDigestPageRewrittenWhole(t *testing.T) {"
  evidence: internal/core/implement/loop/stop.go:824 — "Without a shared folder nothing is written"
  evidence: internal/core/implement/loop/control.go:310 — "noteRunCheckout(home, repoRoot)"
  evidence: internal/core/runstop/runstop.go:810 — "dir := digestDirFor(os.Getuid())"
- ac-5 — MET_WITH_CONCERNS: The walkthrough says what the folder is for, that every account reads it, that it is for a machine no other person shares, and that declining leaves the rest working with a stop reaching only this account; with no setting the stop goes to the account's own home and says its reach. Concerns: the setup page tells the agent to quote what the bare verb prints, while the guard refuses the bare read-only form to an agent (the head's guard blocked it, with or without a folder), so the agent must word the offer from the page instead; and the offer is one section of the page that no install step names, with no gap on the board to prompt it.
  evidence: internal/surface/cli/implement_stop.go:255 — "func sharedFolderWalkthrough() string {"
  evidence: internal/surface/cli/implement_stop.go:261 — "for a machine no other person shares. Declining leaves every other part of abcd working,"
  evidence: internal/core/runstop/runstop.go:642 — "return abcdhome.Path(home), HomeStopFile, nil"
  evidence: internal/surface/cli/implement_stop_test.go:79 — "func TestSharedFolderSaysWhatItIsAndChooses(t *testing.T) {"
  evidence: commands/ahoy.md:1478 — "an install, offer it once, as one question under a `Setup` chip, with what bare"
  evidence: internal/core/guard/personacts.go:91 — "verbs: []string{"ahoy", "shared-folder"}, adjacent: true,"
  evidence: internal/core/ahoy/detect.go:104 — "signals["shared_folder"] = detectSharedFolder()"
- ac-6 — MET: judge opens the path once through OpenRegular and decides on that descriptor: anything but a regular file within the bound owned by the account named at setup is refused and named, the refusal naming the account that wrote it, and no byte is parsed; the refusal reaches the move's answer; the runstop tests pass on the copy.
  evidence: internal/core/runstop/runstop.go:585 — "func judge(p string, want int) (_ *Stop, refused string, unreadable bool) {"
  evidence: internal/core/runstop/runstop.go:602 — "the stop file %s is refused: another account (%s) wrote it, not the person's, so it is no stop"
  evidence: internal/core/runstop/runstop_test.go:189 — "func TestAnotherAccountRefused(t *testing.T) {"
  evidence: internal/core/runstop/runstop_test.go:117 — "func TestContentIsNeverTheRequest(t *testing.T) {"
  evidence: internal/core/implement/loop/stop_test.go:318 — "func TestARefusedFileIsNoStop(t *testing.T) {"

Gap audit:
- honoured:
  - One command from the person's own account stops the runs that read it, at their next move, and it stays until their own command lifts it
    evidence: internal/surface/cli/implement_stop.go:230 — "s, err := runstop.Set(home, now)"
    evidence: internal/core/implement/loop/stop.go:111 — "func stopCheck(repoRoot, runID string, steps Stages, o Options) (string, error) {"
  - A stop is a file another account's run only reads as a request: who and when come from the descriptor, and no byte is read
    evidence: internal/core/runstop/runstop.go:607 — "return &Stop{SetBy: accountName(uid), UID: uid, SetAt: fi.ModTime().UTC(), Path: p}, "", false"
  - An agent in its call finishes, its receipt is collected, and only then is the lane's work saved aside and the claim released
    evidence: internal/core/implement/loop/stop.go:528 — "func collectArrived(repoRoot, runID string, steps Stages, tab *sessionname.Table, o Options) map[string]string {"
    evidence: internal/core/implement/loop/stop.go:653 — "func saveLaneForStop(repoRoot string, root *os.Root, st *State, i int, why string, out *StopLeft, now time.Time) (saved bool) {"
  - A run a stop halted does not step on when the stop lifts; a take-over reserves again first
    evidence: internal/core/implement/loop/terminal_personstop_test.go:243 — "func TestAPersonsHaltWaitsForItsTakeOver(t *testing.T) {"
- diverged:
  - The press release says each run writes what each lane holds uncommitted into its record; the delivery saves it aside as a patch and the record names where
    evidence: internal/core/implement/loop/stop.go:625 — "%d uncommitted file(s) saved aside at %s"
  - The spec names one digest directory in the shared folder; the delivery keeps a directory per account
    evidence: internal/core/runstop/runstop.go:149 — "func digestDirFor(uid int) string { return DigestDir + "-" + strconv.Itoa(uid) }"
  - The setup page has the agent quote what the bare verb prints, and the guard refuses that bare read-only form to an agent
    evidence: commands/ahoy.md:1479 — "`abcd ahoy shared-folder` prints quoted in it"
    evidence: internal/core/guard/defaults/guard.json:1321 — ""abcd-ahoy-shared-folder": {"
  - A first press is said to save and release, and a lane whose runner returned no receipt is kept with its claims and reported as an agent still in its call
    evidence: internal/core/implement/loop/terminal.go:464 — "%s: an agent is still in its call, so the next drive settles what it left"
    evidence: internal/core/implement/loop/stop.go:501 — "if tab == nil || tab.Alive(c.Driver.host()) {"
- missing:
  - The brief invariant the decomposition owes is unwritten, because the ADR it rests on is still proposed and its acceptance is the person's act
    evidence: .abcd/development/decisions/adrs/2611050740453233-a-stop-is-recorded-state-a-person-sets.md:4 — "status: proposed"
    evidence: .abcd/development/decisions/adrs/2611050740453233-a-stop-is-recorded-state-a-person-sets.md:55 — "A brief invariant states this rule once this record is accepted"
  - No test or lab run crosses two real accounts: the second account is a stand-in owner
    evidence: .abcd/development/specs/closed/spc-2611041705580524-from-their-own-account-a-person-types.md:37 — "the tests stand a second account in through the owner the judgement is handed"
  - The daily digest the press release says the stop appears in is a stop-only page, its composer's intent still in drafts
    evidence: .abcd/development/specs/closed/spc-2611041705580524-from-their-own-account-a-person-types.md:24 — "the daily digest (itd-2611041549483696) owns the rest of the page and its composer replaces this one's body"

Scope-condition dispositions:
- cond-2611041705583688 — survived: The check sits in the step a build run takes, in each drain move and in the terminal drive's loop and waits; nothing in the delivery reaches a hosted runner or claims to, and the setting is local to the machine.
  evidence: internal/core/implement/loop/loop.go:994 — "stopNote, err := stopCheck(repoRoot, runID, steps, o)"
  evidence: internal/core/implement/loop/drain.go:442 — "stopNote, serr := drainStop(repoRoot, &st, o)"
  evidence: internal/core/implement/loop/terminal.go:807 — "if t.stopHolds() {"
- cond-2611041705589732 — survived: With no setting the stop stays in the account's home, the board's shared line says a stop reaches only this account's runs, its absence is no gap, and the verb says its reach.
  evidence: internal/core/ahoy/detect.go:1293 — "none chosen: a stop reaches only this account's runs"
  evidence: internal/surface/cli/cli.go:4184 — "if shared, _ := res.Signals["shared_folder"].(string); shared != "" {"
- cond-2611041705586060 — survived: The walkthrough states the assumption and the delivery matches it: stop files and pages are world-readable by mode, a page names no account, and a stop from an account not named at setup is reported and not honoured. abcd does not and cannot check that the machine is unshared.
  evidence: internal/surface/cli/implement_stop.go:261 — "for a machine no other person shares"
  evidence: internal/core/runstop/runstop.go:64 — "sharedFileMode os.FileMode = 0o644"
  evidence: internal/core/runstop/runstop.go:499 — "this account did not name that account at setup, so it is not honoured here"
- cond-2611041705583744 — survived: The stop waits on an agent read as alive or unknown and acts at the next check; the first signal keeps the call's context live; a kill is its own end, which settles nothing and names when the reservation lapses, and the restart path takes the rest.
  evidence: internal/core/implement/loop/stop.go:602 — "live := slices.ContainsFunc(lane.Awaits, func(a Await) bool {"
  evidence: internal/core/implement/loop/stop_test.go:110 — "func TestAStopHoldsAnAgentInItsCall(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_stop_test.go:176 — "func TestASecondSignalKillsTheProcessGroupAndNamesWhenTheReservationLapses(t *testing.T) {"
- cond-2611041705584558 — narrowed: The command is refused to an agent by the guard and by the verb, but the delivery's own records say that holds against an agent acting by mistake only; in one account nothing else stands between an agent and the file.
  narrowing: The refusal holds against an agent acting by mistake; a stop binds an agent only where an operating-system account separates it from the person. Setting a stop also goes ahead where the verb cannot tell whether a host session is above it, leaving the guard as the only refusal there.
  evidence: .abcd/development/decisions/adrs/2611050740453233-a-stop-is-recorded-state-a-person-sets.md:29 — "a stop binds an agent only where an operating-system account separates them"
  evidence: commands/implement.md:893 — "The guard and the verbs hold against an agent acting by mistake"
  evidence: internal/surface/cli/implement_stop.go:60 — "func (c caller) person() bool { return c.terminal && c.look.state != hostSessionFound }"
- cond-2611041705581125 — narrowed: The code has a branch for each platform and no Windows target, but the folder setup suggests on Linux is a temporary tree an age cleaner may empty, which removes a standing stop by no one's command. Only macOS was run for this audit; Linux rests on the code and the pages.
  narrowing: On macOS as stated. On Linux a stop in the suggested folder holds only until the age cleaner reaches it, weeks on; a lasting stop there needs a folder an administrator made elsewhere.
  evidence: internal/core/runstop/runstop.go:137 — "On Linux it is under an age cleaner's care, which the setup's page says."
  evidence: commands/ahoy.md:1493 — "On Linux an age cleaner may remove"
  evidence: internal/core/runstop/runstop_test.go:373 — "func TestTheSuggestedFolderQualifies(t *testing.T) {"
<!-- abcd-review-end receipt=rcp-4a92f9361178 -->
