---
id: spc-2611141617199772
slug: a-drive-carries-on-from-one-command-and
intent: itd-2611141610197753
origin: researcher-authored
production_mode: hand-written
---
# A drive carries on from one command, and says in one line when it cannot

## Summary

spc-2611141617199772 delivers itd-2611141610197753. After an interrupt, `abcd drain --drive` carries on with the lanes its drain lists and nothing else is typed. A lane the drive cannot take is set aside and the next lane opens. One stored state value says `working`, `waiting`, `stalled` or `stopped`, since when, on what and the one act open to a person, and the drive's terminal, `abcd implement status` and the board show it as the same line.

The design is the one the lab on drive operations prototyped and measured (lab-261114145658-67e1f9b, findings F-3 to F-9, F-11 and F-14 to F-17). Its two loop prototypes are 121 changed lines in `control.go` and `terminal.go`; they are the starting point for steps 4 and 1, and they are not merged as they stand. Decisions 1, 2 and 3 of the intent, and the storage part of decision 6, each decided by the run for owner review (dcn-2611141613199819), fix the figures the lab left to the product thinker.

The intent carries five open questions, so it is not ready to build. The steps are ordered so that what needs no answer is built first. Steps 1 and 2, the claim at the start and the interrupt, wait on no question and on no later step, so the fix for iss-2611140902531015 is not held behind the product thinker's answers. Step 3 waits on open question 4, and step 4 on open questions 1 and 2 as well. Steps 5 and 6 follow step 4, as the lab's build order has the state after the set-aside, so they wait on the same questions; the ninth criterion waits on open question 3. This order differs from the lab's brief, which put the lane-less refusal and the set-aside before the claim.

Nothing here adds a command or a flag. No step raises the run-state schema version. Whether the design leaves a stored state a person must clear depends on the intent's open questions 3 and 4.

## Approach

### What is on the default branch today

All paths are under `internal/core/implement/loop/` unless they say otherwise.

- **The driver claim.** `claimDriverMade` (`control.go`) refuses a run whose control record names a live driver that is not this process, with the `contend` at `StageDriver`: "is driven by …, which has been alive since it took the run". `abcd implement resume --take` typed in a terminal records that terminal's login process as the driver. A drive started there is a child of it, reads the run as another session's, and waits (F-3, iss-2611140902531015).
- **The whole save.** `mayWholeSave` (`control.go`) lets a session replace a run file that states an earlier schema version only when the live driver its control record names is the calling process. `haltRun` (`stop.go`) saves through `saveRun`, so a signal's stop on such a run is refused while the recorded driver is the terminal (F-16).
- **The stop's messages.** `stopRefusal`, `waitsForItsTakeOver` and `stopMarkNote` (`stop.go`) compose the paragraph that ends a start on a signal's mark. It names `abcd implement step --run` and `abcd implement resume --take` and tells them apart by whether a stop's mark stands (F-4, F-15).
- **The drive's waits.** `terminal.idle`, `terminal.sleep`, `terminal.failed` and `terminal.limited` (`terminal.go`) are the four waits. `terminal.idleSince` is when the run last stopped moving, and `terminal.moved` clears it. Only `idle` sets it: `failed` and `limited` wait without setting it, and an outage's own clock (`terminalOutageClock.since`) starts when the failures enter the schedule, not at the first failure. `sayWait` prints a wait's sentence when its reason changes and every `terminalSayEvery`. Under `--until` `idle` never ends the run; without it the run ends after `terminalOutageLimit`, eight hours (F-5).
- **The stuck bounds.** `noteRefusal` (`stuck.go`) logs a lane's refusals for `stuckReason`, which hands a lane back after `pace.stuck_refusals` identical refusals or `pace.stuck_minutes` at one stage (`resolveStuckBounds`, 3 and 60 as bundled). It returns at once when the refusal's `Lane` is empty. `runFootprint` (`reservation.go`) refuses with no lane when the issue's record is in no status folder of the checkout, and `reserveOnLeavingLand` calls it at the landing (F-14).
- **Passed lanes.** `terminal.passedLanes` and `DrainOptions.PassedLanes` are a list held in the drive's memory that every later move leaves alone: no save, claim, reservation pushed or renewed, and no agent run (`PassedLane`, `passLane`, `passedLane`, `renewHeld`, `letGo`). A drive whose only lanes left are passed ends `TerminalPassedLanes`.
- **The end of a drive.** `terminal.ended` calls `haltOnSignal` on a stop and then `letGo`, which releases the reservation of each lane in progress unless a stop's record stands on its run. Where `haltRun` was refused, no record stands, so the reservation is released and the run is left with no mark, a driver that is the terminal, and no reservation (F-16).
- **The surfaces.** `driveLog` (`internal/surface/cli/drain_drive.go`) is the one writer of the drive's lines and of its log, `.abcd/.work.local/logs/drain-drive-<stamp>.log`, and `renderDriveEnd` and `driveEndWords` say how a drive ended. `abcd implement status` is rendered in `internal/surface/cli/build.go` and prints each run's record. The board's building line is composed in `internal/core/board/board.go` from `board.Input`, which the root command in `internal/surface/cli/cli.go` builds from `boardOutput` (`internal/surface/cli/board.go`). Its `Status` is the `statusblock.Block` that `boardStatus` (`internal/surface/cli/board_status.go`) reads through `loop.StatusLanes`; with no lane it says "building: nothing right now" (F-17).

### What the lab's prototypes changed

The patches are in the lab's `state/patches/` and on its snapshot's branches `lab/drive-ops-p1` and `lab/drive-ops-p2`, both cut from the pin 67e1f9b8c9b1, which is the default branch these steps start from.

- **P1, second form** (`p1-claim-earlier-runs-2.patch`, 67 lines). In `claimDriverMade`, a live recorded driver is the drive's own when the caller is a drive (`Options.driverIsSelf`), the driver names no session, and `ownAncestor` finds it above the drive in the process table with no session marker on it; the drive then takes the run and the record says the terminal gave way. In `terminal.run`, `takeEarlier` runs before the first move: for each lane of the drain's state that is in progress or parked on its pull request, whose run carries a mark that is not from a person's stop file (`Stop.FromFile` false), it calls `TakeRun`. A person's stop is left alone.
- **P2** (`p2-set-aside.patch`, 54 lines, `terminal.go` only). `terminal.blocked` counts how long one lane has waited on something the drive cannot end, and at the bound adds the lane to `passedLanes` and `DrainOptions.PassedLanes`, says so, and clears `idleSince`. It is called at four places: the contention branch of `driveLane`, `stepRefused` for a refusal that names no lane, `takeOver` at `StageDriver`, and `tookUp`. `moved` clears the count.
- **P4** was not a loop change. It read the drive's sentences back into a state line (`internal/surface/cli/drivescreen/events.go` on the lab's branch) and read them wrong twice (F-11). Nothing of it is taken except the line's fields.

### The landing's reservation refusal (criterion 5; waits on the intent's open question 4)

`noteRefusal` is called for every refusal a step's move returns (`loop.go`, the step's save), for a consumed receipt (`receipt_standing.go`) and from the drive (`terminal.laneRefused`, and the refusal to which `terminal.go` gives the current lane before it logs it). The loop package builds several hundred refusals, and the design review counted over a hundred built without a lane, many of which are not a lane's: the review set (`reviewset.go`), the configuration watch (`configwatch.go`) and the remote's identity (`remoteid.go`) refuse for the machine. Counting every lane-less refusal against the lane in hand would let one machine-wide fault hand every lane back in turn. So this step does not change `noteRefusal`'s rule.

The step is limited to the refusal the lab found. `reserveOnLeavingLand` (`reservation.go`) is called with the lane it lands, and it names that lane on the refusal `runFootprint` returns when the issue's record is in no status folder of the checkout. The existing bounds then count it.

What that costs is stated plainly. With the lane named, the missing-record lane is handed back as stuck after three identical refusals, within three looks. A hand-back as stuck discards the lane's finished and passed work and leaves a mark, a worktree and a branch that a person clears by command. The lab's prototype (F-9) measured the same lane set aside instead, with nothing to clear, and the lab did not build the hand-back. The intent's first scope condition counts a stored state a person must clear against the design, so this step as the lab's brief ordered it is in tension with the intent's own test. That is the intent's open question 4, and this step is not built until it is answered. If the answer is the set-aside, this step is dropped and step 4 gains the prototype's fourth call site, `stepRefused` for a refusal that names no lane and is not a wait by design.

### Set aside and go on (criterion 4; decision 3; waits in part on open questions 1, 2 and 5)

P2's `blocked` is built in `terminal.go` with these changes from the prototype:

- The bound is the stuck bound's minutes, read through `resolveStuckBounds`, not a constant of thirty minutes (decision 3).
- The call sites are the three the stuck bounds never count: a run the drive cannot claim (`takeOver` at `StageDriver`), a run it cannot take up (`tookUp`), and a contention at the lane's opening (`driveLane`). The fourth, `stepRefused` for a lane-less refusal, is built only if open question 4 is answered with the set-aside.
- The words are the set-aside's own. A lane set aside is said as "set aside", with how long it waited and the refusal's reason, and `renderDriveEnd` and `driveEndWords` (`internal/surface/cli/drain_drive.go`) stop saying "since this drive may not save it" for a lane that was set aside for another reason.

Two of the lab's three fixtures no longer reach a set-aside under this plan. The two missing-record fixtures belong to step 3 while open question 4 stands. The own-terminal fixture never reaches a set-aside, because step 1 has the drive claim that run at once. So this step's fixtures are the three call sites above, and the three cases the lab did not try: two lanes blocked at once, a block that clears just after the set-aside, and a drain with a cap of one.

The prototype's set-aside is a passed lane: the drive writes nothing for it, so its reservation is neither released nor renewed, and it is in the drive's memory alone, so only a later start tries it again. Whether either is what ships is open questions 1 and 2. This step builds the prototype's behaviour behind those two questions and asserts neither in a test until each is answered. It does not merge before open questions 1 and 2 are answered: merged as the prototype stands, it would ship the reservation left to lapse, which is the answer the run withdrew. What the end names as the one act when every lane left is set aside is open question 5.

### Claim at the start (criteria 1, 2 and 3; decisions 1 and 2)

P1's second form is built in `control.go` and `terminal.go`:

- `ownAncestor` and the `own` branch of `claimDriverMade`, as prototyped. A live driver in another terminal and an ancestor that a session marker names are still refused (the lab's fixtures four and five).
- `takeEarlier` before the drive's first move, walking the drain's lanes (decision 2). A run the drain no longer lists keeps the note `waitsForItsTakeOver` gives it, said once.
- A person's stop is not taken, lifted or not (decision 1). `passOverHalted` and `waitsForTakeOver` keep their words.
- In `stop.go`, the remedy of `stopRefusal` for a signal's mark and the unsettled branch of `stopMarkNote` no longer send a person at a drive's terminal to two commands. Where the reader is a drive and the run is one its drain lists, the drive takes the run and says "took over"; the paragraph remains for a host session's `abcd implement step`, which is not a drive.
- `TestASignalAtTheArmingStepWaitsForItsTakeOver` (`stop_drive_test.go`) states the old rule and is rewritten. `TestANewDrainNamesARunASignalParked` keeps passing, which is decision 2's edge.

### An interrupt leaves every lane marked or reserved (criterion 10)

After the claim step, a drive that takes its own terminal's run is that run's driver, so `haltRun`'s whole save is the driver's and the stop is applied at any schema version the build reads. One case remains: a lane the drive holds whose run it could not claim. There `haltOnSignal` records, in the drive's memory, each run whose stop could not be applied, and `letGo` and `leftHeld` leave that run's reservation as they already leave one whose stop record stands. The lane then keeps its reservation until its lease lapses, and the next start reads a reserved run, not an unreserved one with no mark.

### The state as a value from the loop (criteria 6 to 9; decisions 3 and 6; criterion 9 waits on open question 3)

The loop computes the state; no surface reads it back out of a sentence.

- **The value.** A new file `drivestate.go` holds `DriveState`: the state word; `Since`; `Until`, absent where the wait names no end; `On`, the issue and its stage or the thing outside the drive; `Do`, one act or nothing; the counts of lanes done, in flight, set aside and handed back; the drive's process id and start time; `Written`; and `Next`, the time by which the drive will have written again. It is set where the loop already knows each field: `sleep` and `idle` (the wait, `idleSince`, the refusal's `Remedy`), `failed` and `limited`, `moved` (working), the set-aside, and `ended` (stopped). A wait's own `Remedy` is used as `Do` only where it is a person's act at a terminal; a remedy written for a host session, such as handing back a receipt, is never shown (F-21).
- **Which waits have a named end.** Two on the default branch: the pace window, whose move names `NextEligibleAt` (`moveWait`, `terminal_wait.go`), and a rate limit whose reset the provider names (`limited`). Every other time the loop holds is the time of its next look, not an end: an outage's next try on its schedule (`terminalOutageClock.fail`), a look at origin (`OriginWait.NextProbeAt`), a look at a cap or at work others hold (`WaitingUntil`), a probe of a limit that names no reset, and `idle`'s look-again. Those waits carry no `Until`. The lab's design also lists a lease's lapse; a wait on another holder's reservation gets an `Until` only if the refusal on the default branch carries that time, which step 5 checks before it sets one.
- **The start of a wait.** The time rule needs to know when the drive last stopped moving, and today only `idle` records that. Step 5 gives `failed` and `limited` their own start: each sets the same time at the first failure, or the first limit, after a movement, if it is not already set, and `moved` clears it. Without this an outage has no `since` to measure from.
- **The stall rule by time.** `stalled` replaces `waiting` when a wait with no `Until` has lasted the stuck bound's minutes since that start, whatever reasons the waits gave in between, which is the prototype's first error written as a test. So an outage reads `waiting` for the stuck bound's minutes and `stalled` from then on, with the `since` of the last movement, which is what the lab's row for the incident note's event 5 and its drawings show ("stalled 2h44m · on outage"). The drive's behaviour does not change.
- **The stall rule by hand-backs.** Not specified. The count, what "the same reason" is keyed on, and whether the drive goes on or pauses are the intent's open question 3. Step 5 builds the time rule and leaves a place for this one.
- **The file.** The drive writes the value whole, by rename, to `.abcd/.work.local/logs/drive-state.json` in its checkout each time a field changes. The file carries a version of its own. It is not run state, and the next drive overwrites it. Nothing was stored in the lab, so the stored form rests on no measurement.
- **Telling a drive that is gone.** A reader that may read the process table checks the recorded process by its id and start time, as `claimDriverMade` and `mayWholeSave` (`control.go`) check a run's recorded driver: `readProcs`, then `sessionname.Table.Alive` on the recorded host, with the same slack the driver's record keeps. A process that is gone reads `stopped`, not heard from since `Written`. `abcd implement status` and the board read this way. This is not the lab's rule in detail: its design says only "a reader that finds the drive's process gone".
- **The next-write time is a fallback.** It is for a reader that may not read the process table, which is the status line of itd-2611141507299828. The drive rewrites the file at every look of a wait, and while an agent is in its call or a preflight runs, at each tick of `renewing` (`terminalRenewTick`, five minutes), which already runs through both; `Next` is the end of the current stretch plus a margin. A stretch of `sleep` can be as long as `terminalRenewEvery`, an hour, so a fallback reader can show a dead drive as `waiting` for up to that long. It never shows it as `working` for longer than one tick and its margin. The seventh criterion is met in full only by a reader of the process table, and the status-line draft's freshness bound is that draft's to set against this.
- **The line.** One function composes the displayed form, `<state> <elapsed> · on <what> · you: <act>`, from the value; where the act is nothing, it reads `nothing to do` while working or waiting and `needs a look` while stalled or stopped.
- **The surfaces.** `internal/surface/cli/drain_drive.go` prints `state: <line>` when the state word, `On` or `Do` changes. `abcd implement status` (`internal/surface/cli/build.go`) prints the line above the runs' records. The board does not get it through its status block: `boardStatus` (`internal/surface/cli/board_status.go`) returns a `statusblock.Block`, which has no field for a drive's state and is also what the site renders. A new reader in `internal/surface/cli/board_drive.go` reads the file; `boardOutput` (`internal/surface/cli/board.go`) gains the value for `--json`; the root command in `internal/surface/cli/cli.go`, which builds `boardOutput` and `board.Input`, hands the composed line to the renderer; and `board.Input` and the building line in `internal/core/board/board.go` show it in place of "nothing right now". A missing, unreadable or newer-version file shows no line and names why on stderr.

### Coordination with the session editing the loop package

Another session is editing `internal/core/implement/loop` this week: `control.go`, `validate.go`, `reviewset.go`, `state.go`, `land.go` and others. So each step below names its files exactly, no step starts until that session's lanes are on the default branch, and each step is cut from the default branch as it then stands. Each step is one pull request. Step 1 edits `control.go`, which that session also edits, and steps 1, 2, 4 and 5 all edit `terminal.go`, so they land one at a time in order.

### The run-state schema

The lab's stop condition forbade a prototype that raised the run-state schema, and none did. No step here raises `SchemaVersion` (26 in `state.go`): the set-aside lives in the drive's memory, the claim adds a line to the run's record through `saveRecord`, which keeps the version the file states, and the drive's state is a file of its own. One risk is stated plainly. Criterion 10 is met for a run the drive could not claim by keeping its reservation, not by writing a mark, because writing the mark's record line is a whole save that would raise an earlier-version file out of its driver's reach. If review asks for the mark on every such run, that needs a save path that does not exist today, and it is a decision to take back to the product thinker and not one to build inside step 2.

## How the Acceptance Criteria are met

The criteria are numbered in the order the intent lists them.

1. A run taken over from the drive's own terminal is taken: `ownAncestor` in `claimDriverMade` (step 1). Proved by the lab's first fixture as a test: three lanes land and the start is the only command.
2. An interrupted drive carries on with every lane its drain lists: `takeEarlier` before the first move, and the `stop.go` messages (step 1). Proved by the lab's second and third fixtures, at this version and at version 25, settled and unsettled; a run the drain no longer lists is named once.
3. Another live session's run and a person's stop are left and named: the `own` branch requires an ancestor that no session marker names, and `takeEarlier` skips `Stop.FromFile` (step 1). Proved by the lab's fixtures four, five and six; that the drive goes on to other work is step 4's.
4. A lane the drive cannot take is set aside at the stuck bound's minutes and the next opens: `terminal.blocked` (step 4). The reservation, the second try and the end's one act wait on open questions 1, 2 and 5.
5. The landing's reservation refusal no longer holds the drive without bound: by the hand-back of step 3 or by the set-aside of step 4, as open question 4 is answered. A machine's refusal is counted against no lane in either case.
6. One line, the same in three places: `DriveState`, its file and the one composing function (steps 5 and 6). Proved by a test that renders the terminal's `state:` line, `abcd implement status` and the board from one stored value and compares the three.
7. A drive that ended without a last line reads `stopped`: the reader's check of the recorded process (step 6).
8. Stalled by time, with the `since` of the last movement, and never before a named end: the rule on the wait's start and `Until` (step 5). Proved by a wait whose reason alternates with the pace window's line, and by an outage that reads `stalled` past the bound.
9. Stalled by lanes handed back in a row: not met by any step until open question 3 is answered.
10. An interrupt leaves every lane marked or reserved: the claim makes the drive the driver, and `letGo` keeps the reservation of a run whose stop could not be applied (step 2). Proved by the lab's interrupt fixture at version 25 and at this version.

## Footprint

- packages: internal/core/implement/loop/reservation.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/control.go, internal/core/implement/loop/stop.go, internal/core/implement/loop/drivestate.go, internal/surface/cli/drain_drive.go, internal/surface/cli/build.go, internal/surface/cli/board_drive.go, internal/surface/cli/board.go, internal/surface/cli/cli.go, internal/core/board/board.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/27-implement.md
- tests: internal/core/implement/loop/stuck_landing_test.go, internal/core/implement/loop/terminal_setaside_test.go, internal/core/implement/loop/terminal_claim_test.go, internal/core/implement/loop/stop_drive_test.go, internal/core/implement/loop/terminal_stop_test.go, internal/core/implement/loop/drivestate_test.go, internal/surface/cli/drain_drive_test.go, internal/surface/cli/drain_drive_state_test.go, internal/surface/cli/board_drive_test.go, internal/surface/cli/board_view_test.go, internal/core/board/board_test.go

## Steps

1. Claim at the start
   - criteria: 1, 2, 3
   - needs: none
   - packages: internal/core/implement/loop/control.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/stop.go
   - tests: internal/core/implement/loop/terminal_claim_test.go: the lab's six fixtures (the driver is the drive's own terminal; an unsettled signal mark on a version 25 run; a run parked on its pull request under a signal mark; a live driver in another terminal; an ancestor that is a host session; a person's stop, lifted). In the fourth and fifth the run is not taken and the drive says who holds it; that the drive then goes on to other work is step 4's to prove. internal/core/implement/loop/stop_drive_test.go: `TestASignalAtTheArmingStepWaitsForItsTakeOver` rewritten to the new rule, and `TestANewDrainNamesARunASignalParked` unchanged and passing. Waits on no open question and on no other step; waits on `control.go` and the other files being free of the other session's lanes.
2. An interrupt leaves every lane marked or reserved
   - criteria: 10
   - needs: 1
   - packages: internal/core/implement/loop/terminal.go
   - tests: internal/core/implement/loop/terminal_stop_test.go: an interrupt to a drive holding a version 25 run whose recorded driver is its terminal applies the stop and saves the lane's work; an interrupt to a drive holding a run it could not claim leaves no mark and keeps the reservation, and the next start reads the run as reserved; the same interrupt at this version is unchanged. Waits on step 1 and on no open question.
3. The landing's reservation refusal names its lane
   - criteria: 5
   - needs: none
   - packages: internal/core/implement/loop/reservation.go
   - tests: internal/core/implement/loop/stuck_landing_test.go: the lab's two missing-record fixtures (under `--until` and with no bound), where the expected result is now that the lane is handed back as stuck after three identical refusals and the two other lanes open and land; a review-set refusal and a configuration refusal that name no lane are logged on no lane and hand nothing back, which is the test of what must not count. Waits on the intent's open question 4, and is dropped if its answer is the set-aside; waits on the other session's lanes.
4. Set aside and go on
   - criteria: 4
   - needs: 1, 3
   - packages: internal/core/implement/loop/terminal.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/implement/loop/terminal_setaside_test.go: a run with a live driver in another terminal, a take-up that is refused, and a contention at a lane's opening are each set aside at the stuck bound's minutes and the two other lanes land; two lanes blocked at once; a block that clears just after the set-aside; a drain with a cap of one ends on its set-aside lane; `pace.stuck_minutes` set in a layer moves the bound. internal/surface/cli/drain_drive_test.go: the end names a set-aside lane with its own reason. No test asserts what becomes of the reservation, whether the same drive tries the lane again, or the end's one act, until the intent's open questions 1, 2 and 5 are answered. Waits on step 1; on step 3, or on open question 4's answer dropping it; and on the intent's open questions 1 and 2, before whose answers this step does not merge.
5. The state as a value from the loop, stored
   - criteria: 8
   - needs: 4
   - packages: internal/core/implement/loop/drivestate.go, internal/core/implement/loop/terminal.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/implement/loop/drivestate_test.go: each wait sets the state, its since and its act; the pace window and a named reset set an until and are never stalled before it; an outage, a look at origin, a cap wait and a limit with no reset set none; an outage and a limit with no reset each have a start of their own, set at the first failure or the first limit after a movement and cleared by the next movement; an outage is waiting short of the stuck bound's minutes from that start and stalled past it; a wait whose reason alternates with the pace window's line is stalled at the bound; a host session's remedy is never the act; the file is written whole with its own version, the drive's process id and start time, and the time of its next write; during an agent's call it is rewritten at each renewal tick. internal/surface/cli/drain_drive_state_test.go: a `state:` line is printed when the state changes and not otherwise. The stall by hand-backs, the intent's ninth criterion, is not built here: it waits on open question 3. Waits on step 4, and so on the questions step 4 waits on.
6. The status surfaces show the line
   - criteria: 6, 7
   - needs: 5
   - packages: internal/surface/cli/build.go, internal/surface/cli/board_drive.go, internal/surface/cli/board.go, internal/surface/cli/cli.go, internal/core/board/board.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/27-implement.md
   - tests: internal/surface/cli/drain_drive_state_test.go: the terminal's `state:` line, `abcd implement status` and the board render the same line from one stored value. internal/surface/cli/board_drive_test.go: a recorded process that is gone reads stopped, not heard from since the value was written, and never working; a missing, unreadable or newer-version file shows no line and names why on stderr. internal/surface/cli/board_view_test.go and internal/core/board/board_test.go: the building line shows the drive's state in place of "nothing right now", and the board's `--json` carries the value. The command page and the two brief pages say what the line holds. Waits on step 5.

## Size

Six pull requests, or five if open question 4 drops step 3. The lab's prototypes put steps 4 and 1 at 54 and 67 changed lines before tests. Step 3 is a few lines. Steps 5 and 6 are new code with no prototype to start from, since the lab's state line was derived from sentences and that approach is not taken.
