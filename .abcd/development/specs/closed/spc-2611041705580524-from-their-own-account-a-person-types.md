---
id: spc-2611041705580524
slug: from-their-own-account-a-person-types
intent: itd-2611041549471712
origin: researcher-authored
production_mode: hand-written
---
# A person stops every unattended run on the machine with one command

## Summary

**What the stop needs by Friday 6 November, and nothing else.** One command from the person's own account, a recorded state every run reads before its next move, the agent in a call finishing, then claims released, then the day's page. Every step is marked `MUST` (for that cut) or `AFTER`. When the `MUST` steps have landed the spec is closed with `--remainder`, which carries the `AFTER` steps into a new open spec and leaves the intent planned until that remainder closes.

**The folder.** Setup is `abcd ahoy shared-folder <folder> [--person <account>]`, run in each account, which in the person's own also makes the directory the stop lives in: it prints what the folder is for, that every account on the machine reads it, and that it is meant for a machine no other person shares, and writes `~/.abcd.noindex/shared-folder.json` (the folder and the person's account). Declining is not running it; `abcd ahoy` then says no stop crosses accounts. The folder must be a real directory owned by root or the person, with each parent held to the same rule; one others can write must carry the sticky bit, so the folder's rule keeps another account from removing what the person wrote (`/Users/Shared` on macOS, `/var/tmp` on Linux, both are).

**The file.** `<folder>/abcd-stop-<number>/abcd-stop.json`, in the person's own directory named by their account's number, or `~/.abcd.noindex/stop.json` where no folder was chosen, which reaches only this account's runs. This is the one file outside the store's owner-only rule (dcn-2611041550139264). It is an input from another account: it is opened once through `fsutil.OpenRegular` (no symlink, no FIFO), judged on that same descriptor (a regular file, at most 4 KiB, owned by the account the person named at setup) and it is a request to stop and nothing more: who and when come from the descriptor, and no byte of it is parsed, printed or acted on. Anything else at that path is refused and named on every move, and the verb that would have set the stop says it did not.

**The check.** The loop's `stopCheck` is the one function every move makes before it moves: it reads the file and, when a stop is in force, applies it to that run and returns a contention refusal at stage `stop` (exit 3). The loop's `haltRun` is the same act for a stop that did not come from the file, which the terminal drive's Ctrl-C joins through the remainder spec. `advanceWith`, `Drive`, `dispatchAwait`, a restart and a run's start call the check, so no lane opens, no stage is handed out and no agent starts; `abcd drain` calls it too and clears a judgement it awaits. `implement receipt` is not refused: an agent's receipt is collected, and the lane goes no further.

**The order (dcn-2611041550139264).** A `stop` entry in the run record naming who set it and when, and an `EventStop` line in the shared log. Then an arrived receipt is collected. A lane with an agent still live is left alone for this move and the next check returns to it. With no agent live, the lane's uncommitted work is saved aside exactly as a restart saves it (a save that fails holds the lane and the claims), an agent lost is marked kept so a restart re-tells it, and only when every lane is past that is the run's reservation released and the run marked halted. Then the day's page is rewritten whole. A pull request already in the merge queue is not disarmed: the stop withdraws what runs do next, and a merge armed before it is the queue's.

**What is alive.** The launch this run recorded, by the process that started it; a step killed outright leaves its agent unknown until the claim's term ends, since the agent leads its own process group; a host's agent is lost once its driver's process is gone (the restart path's record), is in its call while that driver lives, and with no driver recorded is taken to be in its call for one reservation lease. The launch ledger (itd-2611031342011261) replaces that last clause when it lands.

**The page.** `<folder>/abcd-digest/<YYYY-MM-DD>.md`, written atomically by the account that runs the work, naming the stop, every run it halted and what each left. Only that much: the daily digest (itd-2611041549483696) owns the rest of the page and its composer replaces this one's body.

**Who may act.** `abcd implement stop`, its `--lift` and `abcd ahoy shared-folder <folder>` are the person's acts: the guard's rule refuses them to an agent, refuses writes, moves and removals of the stop's files, and the verbs refuse a caller with no terminal or under a host session. In a sticky folder the operating system also refuses another account's removal. The guard and the verbs hold against an agent acting by mistake; a stop binds an agent only where an operating-system account separates them. A lift appends a line to `~/.abcd.noindex/stop-log.jsonl`. The account's home comes from the account database, not the environment. A run pins the folder and the person in its control record at its start and reads that folder, the one the setting names and the platform's shared folder (there for this account, the one the setting names or the one the run started under, what another account leaves being named and not honoured) before each move; a setting that cannot be read, or that has gone or changed during the run, holds the run with nothing let go, as does a home that cannot be found, and only a read that saw clearly and found no stop is a lift or admits a take-over. The chosen folder does not move while a stop stands in it. With no agent left in its call, a run under a stop no longer renews its reservation. A halted run does not step on when the stop lifts: `abcd implement resume --take` (spc-2611041706004822) reserves again first.

How it answers the criteria: the first by the check and the order; the second by the person's acts and the halted run's refusal; the third by `haltRun`, which the terminal drive joins through the remainder spec (itd-2610222215485265 builds the signal); the fourth by the page; the fifth by the setup verb and `commands/ahoy.md`; the sixth by the file's judgement.

## Footprint

- packages: internal/core/runstop/runstop.go, internal/core/implement/loop/stop.go, internal/core/implement/loop/control.go, internal/core/implement/loop/loop.go, internal/core/implement/loop/drive.go, internal/core/implement/loop/restart.go, internal/core/implement/loop/runner_pause.go, internal/core/implement/loop/drain.go, internal/surface/cli/implement_stop.go, internal/surface/cli/implement.go, internal/surface/cli/cli.go, internal/core/ahoy/detect.go, internal/core/guard/personacts.go, internal/core/guard/allowonce.go, internal/core/guard/defaults/guard.json, commands/implement.md, commands/ahoy.md, commands/guard.md, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/27-implement.md, .abcd/development/brief/04-surfaces/17-guard.md
- tests: internal/core/runstop/runstop_test.go, internal/core/implement/loop/stop_test.go, internal/surface/cli/implement_stop_test.go, internal/core/guard/personacts_test.go

## Steps

_Steps 1 and 2 start together; 3 waits for 1; 4 waits for 3 and for spc-2611041706004822's step 1. No step needs a person's login, a key or a real second account: the tests stand a second account in through the owner the judgement is handed._

1. **The folder and the file's judgement** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: none
   - packages: internal/core/runstop/runstop.go
   - tests: internal/core/runstop/runstop_test.go
   - landed: 86776eed1
   The setting, the folder's check, the reader, the set and the lift. Each test watches a refusal: a symlink, a FIFO, an oversized file, a file another account owns, a world-writable folder with no sticky bit.
2. **The person's acts in the guard** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: none
   - packages: internal/core/guard/personacts.go, internal/core/guard/allowonce.go, internal/core/guard/guard.go, internal/core/guard/speculate.go, internal/core/guard/defaults/guard.json, commands/guard.md, .abcd/development/brief/04-surfaces/17-guard.md
   - tests: internal/core/guard/personacts_test.go
   - landed: 787d64b10
   Registry entries and any-program acts for `implement stop` and `ahoy shared-folder`, and a shell write to either file refused.
3. **The check, the order and the page** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: 1
   - packages: internal/core/implement/loop/stop.go, internal/core/implement/loop/control.go, internal/core/implement/loop/loop.go, internal/core/implement/loop/drive.go, internal/core/implement/loop/restart.go, internal/core/implement/loop/runner_pause.go, internal/core/implement/loop/drain.go
   - tests: internal/core/implement/loop/stop_test.go
   - landed: f175c4db5
   `stopCheck`, `haltRun`, the calls, the order above, the page. The first test holds the order: a lane with an agent live keeps its uncommitted work and the reservation stays until a later check.
4. **The verbs and the docs** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: 3
   - packages: internal/surface/cli/implement_stop.go, internal/surface/cli/implement.go, internal/surface/cli/cli.go, internal/core/ahoy/detect.go, commands/implement.md, commands/ahoy.md, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/27-implement.md
   - tests: internal/surface/cli/implement_stop_test.go
   - landed: 205bd50bd
   `abcd implement stop [--lift]`, `abcd ahoy shared-folder`, `abcd ahoy` saying so, the plugin pages and the generated surfaces.
5. **The ADR the trust rule owes** (AFTER)
   - priority: AFTER
   - login: none
   - needs: 3
   - packages: .abcd/development/decisions/adrs
   - tests: none
   The decomposition routes the rule (a stop is a recorded state; lifting it is the person's, by rule; one folder is the one file outside the owner-only rule) to an ADR and a brief invariant, whose acceptance is the person's act. The code holds the rule; this step records it.
6. **A stop over a private network or a cloud adapter** (AFTER)
   - priority: AFTER
   - login: none
   - needs: 3
   - packages: internal/core/runstop
   - tests: internal/core/runstop
   Later work by the ruling (dcn-2611041550139264); not built now.
