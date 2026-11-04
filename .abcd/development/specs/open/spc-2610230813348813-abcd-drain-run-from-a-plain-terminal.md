---
id: spc-2610230813348813
slug: abcd-drain-run-from-a-plain-terminal
intent: itd-2610222215485265
origin: researcher-authored
production_mode: hand-written
---
# abcd-drain-run-from-a-plain-terminal

## Summary

The design record for itd-2610222215485265, planned on the product thinker's sign-off of 2026-10-23 (dcn-2610230813064058). `abcd drain --drive` adds a driving loop. Its rules come from [adr-2610222222326104](../../decisions/adrs/2610222222326104-abcd-drives-agents-with-no-host-session.md), accepted at that sign-off and amended on 2026-11-04, and the intent's decisions as amended the same day. Drafted on 2026-10-22, revised after the Opus and Sonnet review round 1, and revised again on 2026-11-04 so it can be built against the rulings since (dcn-2611031722512037, dcn-2611041550129278, dcn-2611041550139264, dcn-2611041550126639, dcn-2611041608057544), after the first live proof of its walls (dcn-2611031720003633) found the wall has to name what an agent may hold, not what it may not. Every step is test-driven: each test is watched fail first, with a stand-in harness binary on `PATH`, a stand-in runner and a scratch ledger, except the one step marked live.

**What a terminal drain needs by Friday 6 November, and nothing else.** Start; prove the wall; run each stage's agent through the harness as a separate process on this account's subscription; record each receipt; stop; resume. Every step below is marked `MUST` (for that cut) or `AFTER`. When the `MUST` steps have landed, the spec is closed with `--remainder`, which carries the `AFTER` steps into a new open spec and leaves the intent planned until that remainder closes; the intent's changelog line waits with it.

**The route: this account's subscription, on one walled login, and no other.** Every agent abcd starts unattended carries abcd's guard, whichever way its work is paid for (dcn-2611041550129278), so the bare launch the runner uses today, which runs no hooks, is not a launch a drive makes. For the first run each subscription belongs to one account, and an account at 95 per cent of a window waits for its reset rather than moving onto another account's subscription (dcn-2611041608057544). So a drive runs on exactly one walled login per account, and the key route is out of scope: a drive refuses, naming the budget intent (itd-2611011230226453), when the machine's configuration asks for the key route or an `ANTHROPIC_API_KEY` is in its environment. A key on a drive composes with the subscription by the budget intent's rule (dcn-2611041550126639) and runs on a key profile with hooks on (dcn-2611041550129278), once that intent and the profiles intent cover it. Until then the key route is refused, never silently run.

**The walled login, and the seam the profiles intent replaces.** The walled login is one directory under `~/.abcd.noindex/profiles/`, the home the product thinker chose for it (iss-2611032326573729). For this cut there is exactly one such directory on the account: the person makes it and logs the harness into it once, in their own shell (`CLAUDE_CONFIG_DIR=<dir> claude`, then the harness's own login step); abcd never reads, stores or forwards the login token. The directory's settings are abcd's, not the person's: before the proof the drive writes `settings.json` there holding abcd's hooks (the guard, the rules loader and the session hooks) as commands naming, by absolute path, the binary running the drive, so the hooks survive a plugin upgrade and nothing is pinned to a versioned plugin path (iss-2611032326573729), and nothing else. One function answers "the profile this drive runs on" (its directory, the launch's environment and arguments): the drive calls it, and nothing else reads the directory. The fuller model, several named profiles per tool and per way of paying, a creating command, a registry, an allow-list a repository narrows and only the person widens, and a wall re-proved per tool version, is itd-2611040344170206, a later intent, which replaces that one function and moves nothing the person logged in. No profile is built here.

**The wall on the subscription route** is an allow-list: it names what the agent may hold (dcn-2611031720003633). The person's own unmodified `claude -p`, run with `CLAUDE_CONFIG_DIR` naming the walled directory and without `--bare`; `--allowedTools` naming the stage's file and shell tools and nothing else (dcn-2611031722512037), the shell answered by abcd's approver as the runner's launch already does; `--strict-mcp-config` with the approver as the only server; `--no-session-persistence`; `CLAUDE_CODE_DISABLE_CLAUDE_MDS=1` and `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, the repository's rules reaching the agent through the rules loader hook in the walled settings; `ANTHROPIC_API_KEY` sealed out of the environment. Each wall is a check the drive runs before its first move, refusing and naming the wall that failed: the walled directory present and alone, its login state present (as the harness reports it, by state alone, never an account identifier), its settings holding abcd's hooks and nothing else, no key in the environment, and one short live launch in that directory showing the tool list the launch named and the guard refusing a harmless command it is meant to refuse. The live proof runs at the start of a drive, and before each stage the drive compares the walled settings with what it wrote; repeating the live proof while the run goes is the profiles intent's.

**Preconditions, before the first move.** Refuse, naming it, when:
- the runner is not enabled or not admitted;
- there is no walled directory under `~/.abcd.noindex/profiles/`, or more than one (naming them, and itd-2611040344170206 as what admits several);
- the walled directory has no login, or a wall failed its check;
- the configuration asks for the key route, or `ANTHROPIC_API_KEY` is set (naming itd-2611011230226453);
- the judge is routed to opencode;
- a role is pinned to the host.

**Consent, before the first move.** Show the cap (default 8), the estimate (`estimateShares` over the cap), the runner and the route (this account's subscription, on the walled login, naming its directory). Ask once; `--yes` is the scripted answer. Nothing is reserved before the yes.

**The loop.** One move per iteration, under the drain's lock for that move only. After each move:
- **Judging:** the routed judge answers in that move. A judge failure counts toward the outage rule.
- **A lane is open:** drive it in-process exactly as `implement step` does (`loop.Drive` and `DriveReview`, as `build.go` calls them), every role on the walled launch, which is the route the yes named (decision 5). A runner that fails to start a role pauses the lane and a later move retries it, as `Drive` already does (dcn-2611012113432054); the pause counts toward the outage rule. A landing that waits for its receipt runs `abcd implement load --wait -- make preflight` in the lane's worktree, then steps again.
- **The move returned a wait** (the pace window, the cap wait `waiting_until`, or `origin_wait`): sleep until the earliest time named, at least 30 seconds.
- **Outage:** three runner failures in a row, or a judge failure. Wait 1, 5 and 10 minutes, then hourly, and stop after 8 hours. Hand nothing back.

**Stop and resume.** The drive ends at the cap, when nothing is eligible, after an 8-hour outage, or on a stop. Ctrl-C in the terminal is a stop (dcn-2611041608057544), and so are a hang-up and a termination signal: an agent in a call finishes that call, its receipt is recorded, and the drive then releases the lane's reservation and its claims on the work so a peer may take it up (dcn-2611041550139264), leaves the lane on its branch, and prints the run, the lane and the command that resumes it. A second press kills the running process group and exits at once; the reservation then lapses on its own, and the line printed names when. The next `abcd drain --drive` on this copy, or a peer's, takes the lane up from what abcd wrote down. Nothing is handed back for a stop. A stop reaching the machine from elsewhere is itd-2611041549471712; taking a run over from another session is itd-2611041549461012.

## Footprint

- packages: internal/core/runner/claude.go, internal/core/runner/profile.go, internal/core/runner/wall.go, internal/core/implement/loop/terminal.go, internal/core/implement/loop/terminal_wait.go, internal/surface/cli/drain.go, internal/surface/cli/drain_drive.go, internal/surface/cli/drain_drive_signal.go, .abcd/development/release/surface.json, docs/reference/cli/commands.md, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, AGENTS.md
- tests: internal/core/runner/profile_test.go, internal/core/runner/claude_profile_test.go, internal/core/runner/wall_test.go, internal/core/implement/loop/terminal_test.go, internal/core/implement/loop/terminal_wait_test.go, internal/core/implement/loop/terminal_outage_test.go, internal/core/implement/loop/terminal_stop_test.go, internal/surface/cli/drain_drive_test.go, internal/surface/cli/drain_drive_wall_test.go, internal/surface/cli/drain_drive_signal_test.go

## Steps

_Steps 1 and 2 start together; 3 waits for both; 4 and 5 run beside each other once 3 has landed; 6 and 7 follow. Only step 7 needs a person's login, and that person logs in once, in their own shell, into the one walled directory; every other step is built and tested with a stand-in harness binary on `PATH`._

1. **The walled launch and the one profile** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: none
   - packages: internal/core/runner/claude.go, internal/core/runner/profile.go
   - tests: internal/core/runner/profile_test.go, internal/core/runner/claude_profile_test.go
   The claude runner gains a second launch shape beside its bare one: the walled launch of the Summary, with `CLAUDE_CONFIG_DIR` naming the profile's directory, no `--bare`, the allow-list, the approver as the only MCP server, no session persistence, memory and instruction files off, settings read from the walled directory alone with the working tree's excluded by the launch's setting sources, and the key sealed out, through the same `launcher.run` and the same process group as today. `profile.go` is the one function that answers the profile for a drive: the one directory under `~/.abcd.noindex/profiles/`, refusing none or several by name. Each test asserts the argv and the environment the stand-in harness received, and the two refusals.
2. **The terminal loop, against a stand-in runner and a scratch ledger** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: none
   - packages: internal/core/implement/loop/terminal.go, internal/core/implement/loop/terminal_wait.go
   - tests: internal/core/implement/loop/terminal_test.go, internal/core/implement/loop/terminal_wait_test.go, internal/core/implement/loop/terminal_outage_test.go
   The loop of the Summary as a core function, taking its runners through `Runners` so step 1's launch is one value it is handed: one move per iteration under the drain's lock for that move only; a lane driven to its landing in-process through `Drive` and `DriveReview`, every receipt recorded as `Drive` records it; the landing's preflight through `implement load --wait`; each move-named wait slept to the earliest time, never below the floor, the lock free between moves; three failures in a row, or a judge failure, entering the outage schedule, nothing handed back, and the run ending at the cap, at an empty queue, or at 8 hours of outage. A stop request set from outside is honoured at the next call boundary (step 5 wires the signal). The tests drive three seeded issues to their landings on a scratch remote, with a stand-in runner that answers in shape, out of shape, late, and not at all.
3. **The flag, the preconditions and the consent** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: 1, 2
   - packages: internal/surface/cli/drain.go, internal/surface/cli/drain_drive.go, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/surface/cli/drain_drive_test.go
   `--drive` and `--yes` on `abcd drain`, regenerated into the surface record and the command reference. Each precondition of the Summary refuses before anything is reserved, naming what is missing: the key route refuses naming the budget intent, and so does any credential variable the harness would bill through set in the environment; no profile, or several, refuse naming the directory to make or the profiles intent; a wall not yet proved refuses naming it, until step 4 fills that precondition; and a drive started from inside an agent's own session refuses. The consent names the cap, the estimate, the runner and the route; `--drive` with no terminal and no `--yes` refuses; no reservation exists before the yes. Without `--drive` the drain does exactly one move, as today (story 6). The wall checks of step 4 enter here through one precondition the step declares and step 4 fills.
4. **The walled directory's settings and the proof before the first move** (MUST)
   The walled settings are checked against what abcd wrote before each stage starts, and the proof confirms a settings file in the working tree has no effect; a difference stops the drive.
   - priority: MUST, Friday
   - login: none
   - needs: 1, 3
   - packages: internal/core/runner/wall.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/runner/wall_test.go, internal/surface/cli/drain_drive_wall_test.go
   The drive writes the walled directory's `settings.json`, abcd's hooks as absolute commands naming its own binary and nothing else, rewriting it at each start so a new binary re-wires it, and never touching the login state beside it. Then the checks of the Summary, each a wall named in the refusal: the directory present and alone, the login state present, the settings holding abcd's hooks and nothing else, no key in the environment, and the one short live launch showing the tool list the launch named and the guard refusing a command it is meant to refuse. Tested with a stand-in harness that reports each wall held or broken in turn; a settings file the person has edited is replaced and said so.
5. **Stop and resume** (MUST)
   The drive's calls take a context the first signal does not cancel, so the first press lets the call finish; only the second cancels it.
   - priority: MUST, Friday
   - login: none
   - needs: 2, 3
   - packages: internal/surface/cli/drain_drive_signal.go, internal/core/implement/loop/terminal.go
   - tests: internal/surface/cli/drain_drive_signal_test.go, internal/core/implement/loop/terminal_stop_test.go
   The first SIGINT, SIGHUP or SIGTERM is a stop: the runner call in flight finishes, its receipt is recorded, the lane's reservation and claims are released on the scratch remote, the lane stays on its branch, and the exit line names the run, the lane and the command that resumes it. A second signal kills the process group and exits at once, naming when the reservation lapses. A second drive on the same copy, and one on a peer's copy, takes the lane up and lands it (story 5).
6. **Docs** (MUST)
   - priority: MUST, Friday
   - login: none
   - needs: 3, 4, 5
   - packages: commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, AGENTS.md
   - tests: docs-lint and record-lint clean; the command page's fact tests in internal/surface/cli
   `commands/drain.md` (the host's page always calls without `--drive`; the terminal form is the person's way to drive it, dcn-2611041550139264), the brief's drain chapter (its run section says the host drives every move; the drive is the one exception, on the walled login, and the stop's two presses), and AGENTS.md's host-delegated line, which names the ADR from its acceptance and says the exception is not built until this intent ships: it says instead that the drive is built and the key route waits for its budget.
7. **Live proof, on the one walled login** (MUST, live)
   - priority: MUST, Friday
   - login: the person, once, into the one walled directory on the account the drive runs in
   - needs: 4, 5
   - packages: internal/core/runner/wall.go
   - tests: the run's log and the three pull requests on the scratch remote; a dated receipt in the local tier
   A real `abcd drain --drive` on a scratch repository with three seeded minor issues, the real harness on this account's subscription, the cap at 3. The evidence is the run's log, the three pull requests, and the absence of any input after the yes, which is the mechanism's falsifier; one Ctrl-C mid-lane and a resuming drive are part of the run. A wall that does not hold in the real launch is fixed in `wall.go` here or stops the route and goes back to the product thinker. Live; it asks its own yes, since it spends the person's usage and opens real pull requests on a scratch remote. The first proof's lab (dcn-2611031720003633) is the method.
8. **The key route, on a key profile** (AFTER)
   - priority: AFTER
   - login: none
   - needs: 1, 3
   - packages: internal/core/runner/profile.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/runner/profile_test.go, internal/surface/cli/drain_drive_test.go
   Not before itd-2611011230226453 and itd-2611040344170206 ship: a drive whose configuration names a key runs on a key profile with abcd's hooks on, the key reaching the harness by a channel the agent's shell cannot read, under the budget's caps and its composing rule (dcn-2611041550126639, dcn-2611041550129278). Until then the refusal of step 3 stands.
9. **A lane paused past the outage** (AFTER)
   - priority: AFTER
   - login: none
   - needs: 2, 5
   - packages: internal/core/implement/loop/terminal.go
   - tests: internal/core/implement/loop/terminal_outage_test.go
   What the drive does with a lane still paused when its outage runs out: hand it back as stuck with the runner's reason and release it, as the intent's decision 7 was written, or leave it paused on its branch for the next drive, as a stop leaves a lane. Both are on the record; the product thinker settles it before this step is built, and the intent's decision 7 is amended to the answer.

## Size

The `MUST` set is about 25 agent-hours, in two days with steps 1 and 2, then 4 and 5, built beside each other: step 2 and step 4 are the long poles, step 7 is an afternoon on the person's own machine once they have logged in. The `AFTER` set is about a day, after the intents it waits on ship.
