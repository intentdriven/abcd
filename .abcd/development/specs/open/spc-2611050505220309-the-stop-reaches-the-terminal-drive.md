---
id: spc-2611050505220309
slug: the-stop-reaches-the-terminal-drive
intent: itd-2611041549471712
origin: researcher-authored
production_mode: hand-written
---
# the-stop-reaches-the-terminal-drive

## Summary

**What the stop still owes after spc-2611041705580524.** The stop itself has landed: the verb, the file's judgement, the guard's person's acts, the check every move makes, the order it takes, and the day's page. What remains is the ADR the trust rule owes and a stop that reaches the machine from elsewhere; the terminal drive takes part. The third acceptance criterion is answered here: a first press or a hang-up is a stop, which the drive of itd-2610222215485265 takes through the loop's `haltRun` in the same order, from inside the package, waiting for the call in hand; a second press kills the process group, and the drive claims the run with `loop.ClaimDriver`. Each move the drive takes through the loop makes the check. The intent stays planned until this spec closes.

## Footprint

- packages: internal/surface/cli/drain_drive.go, internal/surface/cli/drain_drive_signal.go, internal/core/implement/loop/terminal.go, .abcd/development/decisions/adrs, internal/core/runstop
- tests: internal/surface/cli/drain_drive_signal_test.go, internal/core/implement/loop/stop_test.go, internal/core/runstop/runstop_test.go

## Steps

1. **The ADR the trust rule owes** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: .abcd/development/decisions/adrs
   - tests: none
   The decomposition routes the rule (a stop is a recorded state; lifting it is the person's, by rule; one folder is the one file outside the owner-only rule) to an ADR and a brief invariant, whose acceptance is the person's act. The code holds the rule; adr-2611050740453233 records it, proposed, and what remains of this step is its acceptance and the invariant.
2. **A stop over a private network or a cloud adapter** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: internal/core/runstop
   - tests: internal/core/runstop
   Later work by the ruling (dcn-2611041550139264); not built now.
3. **The terminal drive takes the stop** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: internal/surface/cli/drain_drive.go, internal/surface/cli/drain_drive_signal.go, internal/core/implement/loop/terminal.go
   - tests: internal/surface/cli/drain_drive_signal_test.go
   The first press or a hang-up reaches `haltRun`, which stays inside the package, through the drive's own halting: the running agent's call finishes and its receipt is collected, the lane's uncommitted work is saved aside, its claims are released, and the drive prints the run and the lane; a second press kills the process group at once (dcn-2611041608057544). The drive claims the run through `loop.ClaimDriver`.
