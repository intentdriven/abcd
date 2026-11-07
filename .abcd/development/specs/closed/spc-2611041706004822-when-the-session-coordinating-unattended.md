---
id: spc-2611041706004822
slug: when-the-session-coordinating-unattended
intent: itd-2611041549461012
origin: researcher-authored
production_mode: hand-written
---
# A session that takes over a run steps on from what abcd wrote down

## Summary

**What the restart path needs by Friday 6 November, and nothing else.** A coordinator that dies is replaced by the next session reading recorded state: one listing, one take-over, nothing started twice. Every step is marked `MUST` (for that cut) or `AFTER`. When the `MUST` steps have landed the spec is closed with `--remainder`, which carries the `AFTER` steps into a new open spec and leaves the intent planned until that remainder closes.

**The record.** `.abcd/.work.local/run/<run-id>/control.json`, beside the state and written under the run's lock: the live driver (session, process id, start time) and each in-flight launch with its route, its lane and the process that started it. A file of its own rather than state fields, so no state schema changes while two other branches change that schema, and an older abcd reading the run is unaffected. `loop.ClaimDriver` is the one function a driver calls before it steps: it records the host process the call runs under (`sessionname.FindHost`), or refuses at exit 3 while another driver's process is alive; a driver gone is replaced with an entry in the run record. `abcd implement step` calls it, the terminal drive calls it, and a process table that cannot be read claims nothing and refuses nothing. Each claim also adds the checkout to `~/.abcd.noindex/run-checkouts`, which is how the listing finds a repository other than the one it stands in.

**The listing.** `abcd implement resume` writes nothing and uses no network: every run of this repository's worktrees and of each checkout listed, complete ones left out, each with its driver alive or gone, what a stop left, each lane's stage, each agent alive, lost or unknown, each receipt that arrived uncollected, its pull request, how far the fetched default branch has moved past its base, and how many files it holds uncommitted. A worktree the state names and the loop did not derive is reported, never read.

**The take-over.** `abcd implement resume --take <run-id>`, in the order that makes it safe: refused while a stop holds, and while the process table cannot say whether a recorded driver is gone; claim the driver; collect every receipt that has arrived, before anything else; for each lost agent restart the lane as `step --restart` does (its uncommitted work saved aside and named, its await re-told, one fresh agent per await and none for a receipt collected); for a lane with no agent and uncommitted work, save it aside the same way, a save that fails being a refusal that names no step; if a stop released the reservation, reserve again, refusing at exit 3 with nothing else started when a peer has taken it; clear the halted mark. Starting the fresh agents stays with the next `abcd implement step` and the driver, so the take-over itself spends nothing.

**The drain's judgement.** `finish` already clears `Judging` when a drain ends, which is iss-2610110915210571's remedy; step 2 adds the test its remedy asks for and resolves it.

How it answers the criteria: the first by the record and `ClaimDriver`; the second by the listing and the take-over's order; the third by the aside; the fourth by the drain's test; the fifth because the listing and the checklist are all the take-over reads; the sixth by the stop's mark, the listing's `stop` block and the re-reservation.

## Footprint

- packages: internal/core/implement/loop/control.go, internal/core/implement/loop/resume.go, internal/core/implement/loop/restart.go, internal/core/implement/loop/runner_pause.go, internal/core/occupancy/occupancy.go, internal/surface/cli/implement_resume.go, internal/surface/cli/build.go, internal/surface/cli/implement.go, commands/implement.md, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/27-implement.md
- tests: internal/core/implement/loop/resume_test.go, internal/surface/cli/implement_resume_test.go

## Steps

_Steps 1 and 2 run in order; 3 waits for 2. Nothing here needs a login or a key: the tests stand the process table in through a seam and every agent is a fixture._

1. **The live driver and the launch record** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: none
   - packages: internal/core/implement/loop/control.go, internal/core/implement/loop/runner_pause.go, internal/core/occupancy/occupancy.go, internal/surface/cli/build.go
   - tests: internal/core/implement/loop/resume_test.go
   - landed: f175c4db5
   `ClaimDriver` and its refusal, a launch recorded as its runner is claimed, and the step's call.
2. **The listing, the take-over and the drain's judgement** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: 1
   - packages: internal/core/implement/loop/resume.go, internal/core/implement/loop/restart.go
   - tests: internal/core/implement/loop/resume_test.go
   - landed: f175c4db5
   `ListRuns` and `TakeRun`. The first test holds the order: a receipt at its path is collected and no fresh agent is told for it; a lost agent's uncommitted work is saved aside and named before its await is re-told.
3. **The verb and the docs** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: 2
   - packages: internal/surface/cli/implement_resume.go, internal/surface/cli/implement.go, commands/implement.md, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/27-implement.md
   - tests: internal/surface/cli/implement_resume_test.go
   - landed: 205bd50bd
   `abcd implement resume [--take <run-id>] [--run <run-id>]`, the plugin page and the generated surfaces.
4. **Open pull requests a crash left drifting** (AFTER)
   - priority: AFTER
   - login: none
   - needs: 3
   - packages: internal/core/implement/loop/resume.go
   - tests: internal/core/implement/loop/resume_test.go
   The listing names each pull request from the state without a network read; reading the forge for its queue state (iss-2610241414414823) and sweeping records a crashed copy stranded (iss-2610241107435264) are later.
5. **Each agent's own process, from the launch ledger** (AFTER)
   - priority: AFTER
   - login: none
   - needs: 3
   - packages: internal/core/implement/loop/resume.go
   - tests: internal/core/implement/loop/resume_test.go
   Once itd-2611031342011261 lands, an agent is alive or lost by the process its ledger entry names rather than by the process that started it.
