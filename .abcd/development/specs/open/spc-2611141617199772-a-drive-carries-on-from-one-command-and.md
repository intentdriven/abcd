---
id: spc-2611141617199772
slug: a-drive-carries-on-from-one-command-and
intent: itd-2611141610197753
origin: researcher-authored
production_mode: hand-written
---
# A drive carries on from one command, and says in one line when it cannot

## Summary

spc-2611141617199772 delivers itd-2611141610197753. After an interrupt, a closed terminal, a rebuild or a reboot, `abcd drain --drive` carries on from where the drive stopped with nothing else typed. A lane that cannot go on is set aside and the next lane opens. One stored state value says `working`, `waiting`, `stalled` or `stopped`, since when, on what and the one act open to a person, and the drive's terminal, `abcd implement status` and the board show it as the same line.

The design is the one the lab on drive operations prototyped and measured (lab-261114145658-67e1f9b, findings F-3 to F-9, F-11 and F-14 to F-17). Its two loop prototypes are 121 changed lines in `control.go` and `terminal.go`; they are the starting point for steps 2 and 3, and they are not merged as they stand. The intent's six decisions, each decided by the run for owner review (dcn-2611141613199819), fix the figures and the edges the lab left to the product thinker.

Nothing here adds a command, a flag or a stored state a person must clear. No step raises the run-state schema version.

## Approach

### What is on the default branch today

All paths are under `internal/core/implement/loop/` unless they say otherwise.

- **The driver claim.** `claimDriverMade` (`control.go`) refuses a run whose control record names a live driver that is not this process, with the `contend` at `StageDriver`: "is driven by …, which has been alive since it took the run". `abcd implement resume --take` typed in a terminal records that terminal's login process as the driver. A drive started there is a child of it, reads the run as another session's, and waits (F-3, iss-2611140902531015).
- **The whole save.** `mayWholeSave` (`control.go`) lets a session replace a run file that states an earlier schema version only when the live driver its control record names is the calling process. `haltRun` (`stop.go`) saves through `saveRun`, so a signal's stop on such a run is refused while the recorded driver is the terminal (F-16).
- **The stop's messages.** `stopRefusal`, `waitsForItsTakeOver` and `stopMarkNote` (`stop.go`) compose the paragraph that ends a start on a signal's mark. It names `abcd implement step --run` and `abcd implement resume --take` and tells them apart by whether a stop's mark stands (F-4, F-15).
- **The drive's waits.** `terminal.idle`, `terminal.sleep`, `terminal.failed` and `terminal.limited` (`terminal.go`) are the four waits. `terminal.idleSince` is when the run last stopped moving, and `terminal.moved` clears it. `sayWait` prints a wait's sentence when its reason changes and every `terminalSayEvery`. Under `--until` `idle` never ends the run; without it the run ends after `terminalOutageLimit`, eight hours (F-5).
- **The stuck bounds.** `noteRefusal` (`stuck.go`) logs a lane's refusals for `stuckReason`, which hands a lane back after `pace.stuck_refusals` identical refusals or `pace.stuck_minutes` at one stage (`resolveStuckBounds`, 3 and 60 as bundled). It returns at once when the refusal's `Lane` is empty. `runFootprint` (`reservation.go`) refuses with no lane when the issue's record is in no status folder of the checkout, and `reserveOnLeavingLand` calls it at the landing (F-14).
- **Passed lanes.** `terminal.passedLanes` and `DrainOptions.PassedLanes` are a list held in the drive's memory that every later move leaves alone: no save, claim, reservation pushed or renewed, and no agent run (`PassedLane`, `passLane`, `passedLane`, `renewHeld`, `letGo`). A drive whose only lanes left are passed ends `TerminalPassedLanes`.
- **The end of a drive.** `terminal.ended` calls `haltOnSignal` on a stop and then `letGo`, which releases the reservation of each lane in progress unless a stop's record stands on its run. Where `haltRun` was refused, no record stands, so the reservation is released and the run is left with no mark, a driver that is the terminal, and no reservation (F-16).
- **The surfaces.** `driveLog` (`internal/surface/cli/drain_drive.go`) is the one writer of the drive's lines and of its log, `.abcd/.work.local/logs/drain-drive-<stamp>.log`, and `renderDriveEnd` and `driveEndWords` say how a drive ended. `abcd implement status` is rendered in `internal/surface/cli/build.go` and prints each run's record. The board's building line is composed in `internal/core/board/board.go` from the status block that `internal/surface/cli/board_status.go` reads through `loop.StatusLanes`; with no lane it says "building: nothing right now" (F-17).

### What the lab's prototypes changed

The patches are in the lab's `state/patches/` and on its snapshot's branches `lab/drive-ops-p1` and `lab/drive-ops-p2`, both cut from the pin 67e1f9b8c9b1, which is the default branch these steps start from.

- **P1, second form** (`p1-claim-earlier-runs-2.patch`, 67 lines). In `claimDriverMade`, a live recorded driver is the drive's own when the caller is a drive (`Options.driverIsSelf`), the driver names no session, and `ownAncestor` finds it above the drive in the process table with no session marker on it; the drive then takes the run and the record says the terminal gave way. In `terminal.run`, `takeEarlier` runs before the first move: for each lane of the drain's state that is in progress or parked on its pull request, whose run carries a mark that is not from a person's stop file (`Stop.FromFile` false), it calls `TakeRun`. A person's stop is left alone.
- **P2** (`p2-set-aside.patch`, 54 lines, `terminal.go` only). `terminal.blocked` counts how long one lane has waited on something the drive cannot end, and at the bound adds the lane to `passedLanes` and `DrainOptions.PassedLanes`, says so, and clears `idleSince`. It is called at four places: the contention branch of `driveLane`, `stepRefused` for a refusal that names no lane, `takeOver` at `StageDriver`, and `tookUp`. `moved` clears the count.
- **P4** was not a loop change. It read the drive's sentences back into a state line (`internal/surface/cli/drivescreen/events.go` on the lab's branch) and read them wrong twice (F-11). Nothing of it is taken except the line's fields.

### The lane-less refusal (criterion 5)

`noteRefusal` attributes a refusal with an empty `Lane` to the lane in hand, the run's current lane, and counts it as any other refusal of that lane. The existing bounds then apply unchanged, so a lane whose landing cannot find its record is handed back as stuck after three identical refusals or sixty minutes, with the refusal's own reason. `waitByDesign` refusals stay excluded, as now.

### Set aside and go on (criterion 4; decisions 3, 4 and 5)

P2's `blocked` is built in `terminal.go` with three changes from the prototype:

- The bound is the stuck bound's minutes, read through `resolveStuckBounds`, not a constant of thirty minutes (decision 3).
- The call in `stepRefused` for a lane-less refusal is not built. After the first step the stuck bounds count that refusal, and the lane is handed back as today. The set-aside covers what the stuck bounds do not count: a run the drive cannot claim (`takeOver` at `StageDriver`), a run it cannot take up (`tookUp`), and a contention at the lane's opening (`driveLane`).
- The words are the set-aside's own. A lane set aside is said as "set aside", with how long it waited and the refusal's reason, and `renderDriveEnd` and `driveEndWords` (`internal/surface/cli/drain_drive.go`) stop saying "since this drive may not save it" for a lane that was set aside for another reason. `PassedLane` gains nothing a person clears; the reason it already carries says which kind it is.

The drive writes nothing for a set-aside lane from then on, so its reservation is neither released nor renewed (decision 4), and it is in the drive's memory alone, so the next start tries the lane again (decision 5). The three cases the lab did not try are tests of this step: two lanes blocked at once, a block that clears just after the set-aside, and a drain with a cap of one.

### Claim at the start (criteria 1, 2 and 3; decisions 1 and 2)

P1's second form is built in `control.go` and `terminal.go`:

- `ownAncestor` and the `own` branch of `claimDriverMade`, as prototyped. A live driver in another terminal and an ancestor that a session marker names are still refused (the lab's fixtures four and five).
- `takeEarlier` before the drive's first move, walking the drain's lanes (decision 2). A run the drain no longer lists keeps the note `waitsForItsTakeOver` gives it, said once.
- A person's stop is not taken, lifted or not (decision 1). `passOverHalted` and `waitsForTakeOver` keep their words.
- In `stop.go`, the remedy of `stopRefusal` for a signal's mark and the unsettled branch of `stopMarkNote` no longer send a person at a drive's terminal to two commands. Where the reader is a drive and the run is one its drain lists, the drive takes the run and says "took over"; the paragraph remains for a host session's `abcd implement step`, which is not a drive.
- `TestASignalAtTheArmingStepWaitsForItsTakeOver` (`stop_drive_test.go`) states the old rule and is rewritten. `TestANewDrainNamesARunASignalParked` keeps passing, which is decision 2's edge.

### An interrupt leaves every lane marked or reserved (criterion 10)

After the claim step, a drive that takes its own terminal's run is that run's driver, so `haltRun`'s whole save is the driver's and the stop is applied at any schema version the build reads. One case remains: a lane the drive holds whose run it could not claim. There `haltOnSignal` records, in the drive's memory, each run whose stop could not be applied, and `letGo` and `leftHeld` leave that run's reservation as they already leave one whose stop record stands. The lane then keeps its reservation until its lease lapses, and the next start reads a reserved run, not an unreserved one with no mark.

### The state as a value from the loop (criteria 6 to 9; decision 6)

The loop computes the state; no surface reads it back out of a sentence.

- **The value.** A new file `drivestate.go` holds `DriveState`: the state word; `Since`; `Until`, absent where the wait names no end; `On`, the issue and its stage or the thing outside the drive; `Do`, one act or nothing; the counts of lanes done, in flight, set aside and handed back; `Written`; and `Next`, the time by which the drive will have written again. It is set where the loop already knows each field: `sleep` and `idle` (the wait, `idleSince`, the refusal's `Remedy`), `failed` and `limited` (an outage's or a limit's end), `moved` (working), the set-aside, and `ended` (stopped, with `TerminalResumeCommand` as the act where a later drive carries on). A wait's own `Remedy` is used as `Do` only where it is a person's act at a terminal; a remedy written for a host session, such as handing back a receipt, is never shown (F-21).
- **The stall rules.** `stalled` replaces `waiting` when a wait with no `Until` has lasted the stuck bound's minutes since `idleSince`, whatever reasons the waits gave in between, which is the prototype's first error written as a test. It also replaces `working` when as many lanes in a row as the stuck bound's refusals were handed back as stuck at the same stage and check; the count is kept in the drive's memory and a lane that lands clears it. The drive's behaviour does not change on either.
- **The file.** The drive writes the value whole, by rename, to `.abcd/.work.local/logs/drive-state.json` in its checkout each time a field changes and at each look of a wait. The file carries a version of its own. It is not run state, and the next drive overwrites it.
- **The line.** One function composes the displayed form, `<state> <elapsed> · on <what> · you: <act>`, from the value; where the act is nothing, it reads `nothing to do` while working or waiting and `needs a look` while stalled or stopped. A reader that finds the value past its `Next` composes `stopped`, not heard from since `Written`, without reading the process table.
- **The surfaces.** `internal/surface/cli/drain_drive.go` prints `state: <line>` when the state word, `On` or `Do` changes. `abcd implement status` (`internal/surface/cli/build.go`) prints the line above the runs' records. The board (`internal/core/board/board.go`, fed by `internal/surface/cli/board_status.go`) prints it on its building line in place of "nothing right now" when a drive's state is present. A missing, unreadable or newer-version file shows no line and names why on stderr.

### Coordination with the session editing the loop package

Another session is editing `internal/core/implement/loop` this week: `control.go`, `validate.go`, `reviewset.go`, `state.go`, `land.go` and others. So each step below names its files exactly, no step starts until that session's lanes are on the default branch, and each step is cut from the default branch as it then stands. Each step is one pull request. Step 3 edits `control.go`, which that session also edits, and steps 2 to 5 all edit `terminal.go`, so they land one at a time in order.

### The run-state schema

The lab's stop condition forbade a prototype that raised the run-state schema, and none did. No step here raises `SchemaVersion` (26 in `state.go`): the set-aside and the hand-back count live in the drive's memory, the claim adds a line to the run's record through `saveRecord`, which keeps the version the file states, and the drive's state is a file of its own. One risk is stated plainly. Criterion 10 is met for a run the drive could not claim by keeping its reservation, not by writing a mark, because writing the mark's record line is a whole save that would raise an earlier-version file out of its driver's reach. If review asks for the mark on every such run, that needs a save path that does not exist today, and it is a decision to take back to the product thinker and not one to build inside step 4.

## How the Acceptance Criteria are met

The criteria are numbered in the order the intent lists them.

1. A run taken over from the drive's own terminal is taken: `ownAncestor` in `claimDriverMade` (step 3). Proved by the lab's first fixture as a test: three lanes land and the start is the only command.
2. An interrupted drive carries on with every lane its drain lists: `takeEarlier` before the first move, and the `stop.go` messages (step 3). Proved by the lab's second and third fixtures, at this version and at version 25, settled and unsettled; a run the drain no longer lists is named once.
3. Another live session's run and a person's stop are left and named: the `own` branch requires an ancestor that no session marker names, and `takeEarlier` skips `Stop.FromFile` (step 3). Proved by the lab's fixtures four, five and six.
4. A lane that cannot go on is set aside at the stuck bound's minutes and the next opens: `terminal.blocked` (step 2). Proved by the lab's three fixtures and the three it did not try.
5. A refusal that names no lane is counted: `noteRefusal` (step 1). Proved by a landing whose record is missing being handed back at the stuck bounds.
6. One line, the same in three places: `DriveState`, its file and the one composing function (steps 5 and 6). Proved by a test that renders the terminal's `state:` line, `abcd implement status` and the board from one stored value and compares the three.
7. A drive that ended without a last line reads `stopped`: the reader's rule on `Next` (step 6).
8. Stalled by time, with the `since` of the last movement: the rule on `idleSince` (step 5). Proved by a wait whose reason alternates with the pace window's line.
9. Stalled by lanes handed back in a row: the count by stage and check (step 5).
10. An interrupt leaves every lane marked or reserved: the claim makes the drive the driver, and `letGo` keeps the reservation of a run whose stop could not be applied (step 4). Proved by the lab's interrupt fixture at version 25 and at this version.

## Footprint

- packages: internal/core/implement/loop/stuck.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/control.go, internal/core/implement/loop/stop.go, internal/core/implement/loop/drivestate.go, internal/surface/cli/drain_drive.go, internal/surface/cli/build.go, internal/surface/cli/board_status.go, internal/core/board/board.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/27-implement.md
- tests: internal/core/implement/loop/stuck_laneless_test.go, internal/core/implement/loop/terminal_setaside_test.go, internal/core/implement/loop/terminal_claim_test.go, internal/core/implement/loop/stop_drive_test.go, internal/core/implement/loop/terminal_stop_test.go, internal/core/implement/loop/drivestate_test.go, internal/surface/cli/drain_drive_test.go, internal/surface/cli/drain_drive_state_test.go, internal/surface/cli/board_status_test.go, internal/core/board/board_test.go

## Steps

1. The lane-less refusal is counted
   - criteria: 5
   - needs: none
   - packages: internal/core/implement/loop/stuck.go
   - tests: internal/core/implement/loop/stuck_laneless_test.go: a refusal with no lane from `runFootprint` at the landing is logged on the lane in hand; the lane is handed back as stuck after three identical refusals and after sixty minutes at one stage; a `waitByDesign` refusal with no lane is still not counted. Waits on the other session's lanes.
2. Set aside and go on
   - criteria: 4
   - needs: 1
   - packages: internal/core/implement/loop/terminal.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/implement/loop/terminal_setaside_test.go: the lab's three fixtures (the own-terminal driver with no bound, the missing record under `--until` and with no bound) each set the lane aside at the stuck bound's minutes and land the two other lanes; two lanes blocked at once; a block that clears just after the set-aside stays set aside until the next start; a drain with a cap of one ends on its set-aside lane; the reservation of a set-aside lane is neither released nor renewed; `pace.stuck_minutes` set in a layer moves the bound. internal/surface/cli/drain_drive_test.go: the end names a set-aside lane with its own reason. Waits on step 1.
3. Claim at the start
   - criteria: 1, 2, 3
   - needs: 2
   - packages: internal/core/implement/loop/control.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/stop.go
   - tests: internal/core/implement/loop/terminal_claim_test.go: the lab's six fixtures (the driver is the drive's own terminal; an unsettled signal mark on a version 25 run; a run parked on its pull request under a signal mark; a live driver in another terminal; an ancestor that is a host session; a person's stop, lifted). internal/core/implement/loop/stop_drive_test.go: `TestASignalAtTheArmingStepWaitsForItsTakeOver` rewritten to the new rule, and `TestANewDrainNamesARunASignalParked` unchanged and passing. Waits on step 2, and on `control.go` being free of the other session's lanes.
4. An interrupt leaves every lane marked or reserved
   - criteria: 10
   - needs: 3
   - packages: internal/core/implement/loop/terminal.go
   - tests: internal/core/implement/loop/terminal_stop_test.go: an interrupt to a drive holding a version 25 run whose recorded driver is its terminal applies the stop and saves the lane's work; an interrupt to a drive holding a run it could not claim leaves no mark and keeps the reservation, and the next start reads the run as reserved; the same interrupt at this version is unchanged. Waits on step 3.
5. The state as a value from the loop, stored
   - criteria: 8, 9
   - needs: 4
   - packages: internal/core/implement/loop/drivestate.go, internal/core/implement/loop/terminal.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/implement/loop/drivestate_test.go: each of the four waits sets the state, its since, its until and its act; a wait with an until is never stalled before it; a wait whose reason alternates with the pace window's line is stalled at the stuck bound's minutes with the since of the last movement; three lanes handed back as stuck at one stage and check read stalled, and a landed lane clears the count; a host session's remedy is never the act; the file is written whole with its own version and the time of its next write. internal/surface/cli/drain_drive_state_test.go: a `state:` line is printed when the state changes and not otherwise. Waits on step 4.
6. The status surfaces show the line
   - criteria: 6, 7
   - needs: 5
   - packages: internal/surface/cli/build.go, internal/surface/cli/board_status.go, internal/core/board/board.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/27-implement.md
   - tests: internal/surface/cli/drain_drive_state_test.go: the terminal's `state:` line, `abcd implement status` and the board render the same line from one stored value; a value past its next write reads stopped, not heard from since it was written, and never working; a missing, unreadable or newer-version file shows no line and names why on stderr. internal/surface/cli/board_status_test.go and internal/core/board/board_test.go: the building line shows the drive's state in place of "nothing right now". The command page and the two brief pages say what the line holds. Waits on step 5.

## Size

Six pull requests. The lab's prototypes put steps 2 and 3 at 54 and 67 changed lines before tests. Step 1 is a few lines. Steps 5 and 6 are new code with no prototype to start from, since the lab's state line was derived from sentences and that approach is not taken.
