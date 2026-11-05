---
id: spc-2611050505220309
slug: the-stop-reaches-the-terminal-drive
intent: itd-2611041549471712
origin: researcher-authored
production_mode: hand-written
---
# the-stop-reaches-the-terminal-drive

## Summary

**What the stop still owes after spc-2611041705580524.** The stop itself has landed: the verb, the file's judgement, the guard's person's acts, the check every move makes, the order it takes, and the day's page. What remains is the terminal drive taking part, the ADR the trust rule owes, and a stop that reaches the machine from elsewhere. The third acceptance criterion is answered here: the loop's `haltRun` takes a stop that did not come from the file through the same order and is tested, and nothing outside the package calls it until the drive of itd-2610222215485265 does, so Ctrl-C in a terminal run is not yet a stop. The drive's own signal handling lives on its own branch, and this spec joins the two when both have landed: the loop exports `HaltRun` over `haltRun` in the change that calls it, a first press or a hang-up calls it and the drive waits for the call in hand, a second kills the process group, and the drive claims the run with `loop.ClaimDriver`. Each move the drive takes through the loop already makes the check. The intent stays planned until this spec closes.

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
   The decomposition routes the rule (a stop is a recorded state; an agent's session may never lift it; one folder is the one file outside the owner-only rule) to an ADR and a brief invariant, whose acceptance is the person's act. The code holds the rule; adr-2611050740453233 records it, proposed, and what remains of this step is its acceptance and the invariant.
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
   Once itd-2610222215485265 lands: the loop exports `HaltRun` over `haltRun`, the first press or a hang-up calls it, the running agent's call finishes and its receipt is collected, the lane's uncommitted work is saved aside, its claims are released, and the drive prints the run and the lane; a second press kills the process group at once (dcn-2611041608057544). The drive claims the run through `loop.ClaimDriver`.
