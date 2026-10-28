---
id: spc-2610230813348813
slug: abcd-drain-run-from-a-plain-terminal
intent: itd-2610222215485265
origin: researcher-authored
production_mode: hand-written
---
# abcd-drain-run-from-a-plain-terminal

## Summary

The design record for itd-2610222215485265, planned on the product thinker's sign-off of 2026-10-23 (dcn-2610230813064058). `abcd drain --drive` adds a driving loop. Its rules come from [adr-2610222222326104](../../decisions/adrs/2610222222326104-abcd-drives-agents-with-no-host-session.md), accepted at that sign-off, and the intent's decisions; the route a drive runs on is the product thinker's ruling of 2026-10-23 (dcn-2610230716408975): the person's subscription by default, walled, or an API key when their configuration asks for it. Drafted on 2026-10-22 and revised after the Opus and Sonnet review round 1. Every step is test-driven: each test is watched fail first, with a stand-in runner and a scratch ledger, except the steps marked live.

**The wall on the subscription route.** The person's own unmodified `claude -p`, run with a dedicated `CLAUDE_CONFIG_DIR` the person logs into once, `CLAUDE_CODE_DISABLE_CLAUDE_MDS=1`, `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, `--strict-mcp-config` with an empty `--mcp-config`, `--settings '{"disableAllHooks":true}'` and `--no-session-persistence`, with `ANTHROPIC_API_KEY` unset. abcd never reads, stores or forwards the login token. The key route keeps `--bare`. Each wall is a check the drive runs before its first move.

**Preconditions, before the first move.** Refuse, naming it, when:
- the runner is not enabled or not admitted;
- on the subscription route, the walled configuration directory has no login or a wall failed its live check; on the key route, there is no API key;
- the judge is routed to opencode;
- a role is pinned to the host.

**Consent, before the first move.** Show the cap (default 8), the estimate (`estimateShares` over the cap), and the runner and route. Ask once; `--yes` is the scripted answer. Nothing is reserved before the yes.

**The loop.** One move per iteration, under the drain's lock for that move only. After each move:
- **Judging:** the routed judge answers in that move. A judge failure counts toward the outage rule.
- **A lane is open:** drive it in-process exactly as `implement step` does (`loop.Drive` and `DriveReview`, as `build.go` calls them), with `fallback_host` serving unrouted roles. A `Fallback` result, a role left awaiting a host, becomes the existing stuck hand-back with the runner's reason, plus the reservation's release. A landing that waits for its receipt runs `abcd implement load --wait -- make preflight` in the lane's worktree, then steps again.
- **The move returned a wait** (the pace window, the cap wait `waiting_until`, or `origin_wait`): sleep until the earliest time named, at least 30 seconds.
- **Outage:** three runner failures in a row, or a judge failure. Wait 1, 5 and 10 minutes, then hourly, and stop after 8 hours. Hand nothing back.

**Stop.** At the cap, when nothing is eligible, after an 8-hour outage, or on a signal. The first SIGINT or SIGHUP kills the running runner's process group, leaves the lane resumable, and prints the run, the lane and when the reservation lapses; the second exits at once.

## Footprint

- packages: internal/surface/cli, internal/core/implement/loop, internal/core/runner, commands/drain.md, .abcd/development/brief/04-surfaces
- tests: internal/surface/cli, internal/core/implement/loop, internal/core/runner

## Steps

1. **Prove the walls live, before anything else is built**
   On a real machine with the person's own setup in place (a CLAUDE.md, auto-memory, an MCP server, a claude.ai connector, a user hook and a plugin hook, saved sessions), run the walled `claude -p` and show that none of them reaches the agent. Prove the three edges the docs leave silent: `settings.json` under `--bare`, claude.ai connectors under `--strict-mcp-config`, and plugin hooks under `disableAllHooks`. Each wall becomes a check the drive runs before its first move, refusing and naming the wall that failed. If a wall cannot be made to hold, the subscription route stops here and goes back to the product thinker. Live; it asks its own yes, since it spends the person's usage.
   - packages: internal/core/runner
   - tests: internal/core/runner
2. **Preconditions and the flag**
   Each refusal fires before anything is reserved; without `--drive` the drain does exactly one move, as today (story 6).
   - packages: internal/surface/cli, internal/core/runner
   - tests: internal/surface/cli
3. **Consent**
   No reservation before the yes; `--drive` with no terminal and no `--yes` refuses; the prompt names the cap, the estimate and the route (story 1).
   - packages: internal/surface/cli
   - tests: internal/surface/cli
4. **The in-process lane driver**
   Each test drives a lane to its landing with every role routed. A `Fallback` becomes a stuck hand-back and the reservation is released (a tombstone on a scratch remote); no lane is left awaiting (stories 2 and 3).
   - packages: internal/core/implement/loop
   - tests: internal/core/implement/loop
5. **The driver's preflight**
   A lane waiting for its receipt gets one, minted through the load gate, and lands.
   - packages: internal/core/implement/loop
   - tests: internal/core/implement/loop
6. **Waits and outage**
   Each move-named wait sleeps until the right time, never below the floor; the lock is free between moves; three failures enter the outage schedule, nothing is handed back, and it stops at 8 hours (story 4).
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: internal/core/implement/loop
7. **Signals**
   SIGINT and SIGHUP stop the runner group and leave the lane resumable; a second drive resumes the lane; a second signal exits at once (story 5).
   - packages: internal/surface/cli
   - tests: internal/surface/cli
8. **Live proof**
   A real `--drive` on a scratch repository with three seeded minor issues and the real claude runner, with the cap at 3. The evidence is the run's log, the three pull requests, and the absence of any input after the yes, which is the mechanism's falsifier. Live; it asks its own yes, since it spends the person's usage and opens real pull requests on a scratch remote.
   - packages: internal/surface/cli
   - tests: internal/surface/cli
9. **Docs**
   `commands/drain.md` (the host page always calls without `--drive`), the brief's drain chapter and the generated docs; AGENTS.md's host-delegated line, which names the ADR from its acceptance, drops "not built".
   - packages: commands/drain.md, .abcd/development/brief/04-surfaces, AGENTS.md
   - tests: internal/surface/cli

## Size

About 7 to 8 days. Steps 1, 4, 5 and 8 are the long poles.
