---
name: implement
description: Share one autonomous run between two sessions — join it, open a window in a division mode, claim a record before opening its lane, check the second session's bounds, log the run's events, and derive the comparison of the modes, and check the machine's load before abcd's own tests start — by invoking the abcd binary. The bare form and report are read-only; the load check warns and never refuses.
argument-hint: "[join|leave|mode|claim|release|check|log|report|load] …"
---

# `/abcd:implement` — share a run between sessions

An autonomous run is one session's by default. A second session may join it
for a window, and the run divides the work one of three ways — a claim per
record, whole batches per session, or the first building while the second
reviews and lands — and measures which way worked. This page is the run
machinery a driving session calls; it is not the verb a person types to build
an intent.

Everything lives in the machine-scoped run state, `~/.abcd/runs/<root-sha>/`,
keyed on the repository's root commit, so sessions in different worktrees of
one repository share one run and no repository file. Nothing here ever writes
to the checkout.

## See where the run stands

Bare invocation is read-only and creates nothing:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement --json
```

Report the `window` (the division mode in force, or none), the joined
`sessions` with their roles, and the `claims` — each with its holder, its lane,
when its lease ends, and whether it is still `live`.

## Join, and open a window

Every write acts for a joined session, named with `--session` on every call.
Join first, stating the role:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement join --session <id> --role first|second [--ceiling <n>] --json
```

Joining writes the session's record and a `session_open` line, and signals no
one: the first session learns of a second only by reading the run state, and
never waits on it. Joining again with the same role is a resume; asking for the
other role is refused.

A second session states its own agent ceiling with `--ceiling`: the most agents
it runs at once, kept on top of the first session's, never instead of it. abcd
runs and counts no agent, so the ceiling is the session's own discipline: it is
recorded, carried on the `session_open` line, and reported by every `check`
(`ceiling` in the verdict), and a resume cannot restate it. Before starting an
agent, the second session counts its own running agents against it and, at the
ceiling, waits and logs a `ceiling_wait`.

The first session opens each window by naming its mode — `single`, `claim`,
`batch` or `split-roles`:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement mode claim --session <id> --window <n> --json
```

The second session cannot set the mode.

## Claim a record before opening its lane

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement claim <record> --session <id> --lane <lane> [--lease 2h] [--path <file> …] --json
```

One claim file per record, created exclusively: of two sessions reaching for
one record, exactly one holds it. A claim is a lease (default two hours); a
session that dies holding one strands nothing, because a lapsed lease is
claimable again and the lapse is logged. A claim file nobody can parse (a session
killed mid-claim) holds its record for one minute from when it was written, then
lapses the same way, logged with reason `unparseable`. Claiming a record this
session already holds renews the lease. Release it when the lane is done, and leave when the
session stops:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement release <record> --session <id> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" implement leave --session <id> --reason "<why>" --json
```

**Exit 3 means back off.** A record another session holds is refused at exit 3,
naming the holder, and logged as `claim_denied` (the second session also logs a
`backoff`). Take other work; do not retry the same record in a loop. A locked
run state is the same exit.

## The second session's bounds

The second session is refused at exit 2, and the refusal is logged, when it:

- claims while it already holds a live claim — one lane at a time;
- claims in a `split-roles` window — there it reviews, audits and lands only;
- declares a `--path` in the reading corpus — every position's `object.paths` in
  the committed `.abcd/config/reading-presets.json`, plus that file — those
  lanes are the first's; when the preset file is absent or unreadable, any
  declared `--path` is refused, since nothing can say the lane is clear;
- reaches the release step — only the first session cuts a release.

It also keeps its own agent ceiling (stated on joining, reported by `check`),
which no verb here enforces.

Before a step that is not a claim, ask:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement check release|lane|review|audit|land --session <id> [--path <file> …] --json
```

An allowed step writes nothing. On a refusal, stop that step and leave it to the
first session; a stop condition the second session meets stops only itself.

## Log the run's events

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement log <event> --session <id> --field key=value … --json
```

One line per event, appended in a single write, so two sessions writing at once
each land whole lines. The events are `backoff`, `lane_open`, `lane_close`,
`agent_start`, `agent_end`, `ceiling_wait`, `gate_run`, `review`, `fallback`,
`stop`, `refusal`, `pr`, `capture` and `context`. For the comparison to count
them: a `lane_close` with `outcome=merged` (or `landed`) is a lane landed;
`backoff` and `ceiling_wait` carry `minutes`, and `agent_end` carries `minutes`,
`wall_minutes` or `wall_min`; a `context` line carries `used_pct` (with `role`
and `note`), the orchestrator's share of its context window in use. The session, window and claim
events belong to their own sub-verbs and are refused here.

## Compare the modes

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement report [--date YYYY-MM-DD | --log <file>] --json
```

Read-only. Per mode: windows, wall clock, lanes opened and landed, the second
session's lanes landed, collisions, lapsed claims, backoffs and the minutes
backed off, agent minutes, ceiling wait and refusals, with each session's share;
and, per session across the run, its context lines and the last `used_pct` seen.
`leader` is the mode with the most lanes landed per wall-clock hour — a figure,
not a verdict: the run's own report names the mode it would keep and says why.
Relay any `unparsed` lines; they are counted nowhere.

## Check the machine's load

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement load --site preflight|eval-harness [--json]
```

`make preflight` runs this first and the eval harness runs it once at its
start, so it rarely needs calling by hand. It reads the machine's load and
process table once and warns about two things: a program outside the running
work that has used nearly all the CPU it could get for longer than the stray
limit (30 minutes by default), and a one-minute load average above the extreme
limit (four times the online core count by default). What a program could get
is its fair share: the online cores divided by the one-minute load, never more
than one core. So on a loaded machine a stray can show well under 100% of a
core: forty busy loops on 16 cores each show about 40%, and all forty are
strays. The person's programs and other accounts' are judged alike. It never
refuses, never waits and never stops anything; it exits 0 on every `status`: `ok`, `warning`, `skipped` (in CI, with
the `reason`) and `unchecked` (a platform other than macOS and
Linux, or a read that failed, with the `reason`).

Relay a `warning` whole. The person's own strays (`own_strays`) are named with
pid, process group, age and CPU share, and `remedy` gives, per stray or per
wholly-stray group, a re-check to run before each kill: `pgrep -g <group>` must
list only the named pids before `kill -- -<group>`, and `ps -o pid=,comm= -p
<pid>` must still show the program before `kill <pid>`. Never stop anything
yourself, and never by pattern (`pkill -f`, `killall`): the choice to stop is
the person's. Other accounts' strays (`other_strays`) are only a count and a CPU
total; say nothing more about them. A name the private banned-names layer
matches reads `[private name]`.

The limits are per machine, in `~/.abcd/load-limits`, which the check reads and
never creates: `stray-minutes <1 to 10080>` and `extreme-load <load>`, one per
line, `#` for comments. An unusable file is reported (`limits.malformed`) and
both defaults are used. Inside an autonomous run, a warning is also written to
the run log as a `load` event (`run_log`).

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
