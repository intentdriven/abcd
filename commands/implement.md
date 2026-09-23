---
name: implement
description: Share one autonomous run between two sessions — join it, open a window in a division mode, claim a record before opening its lane, check the second session's bounds, log the run's events, and derive the comparison of the modes — by invoking the abcd binary. The bare form and report are read-only.
argument-hint: "[join|leave|mode|claim|release|check|log|report] …"
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
"${CLAUDE_PLUGIN_ROOT}/abcd" implement join --session <id> --role first|second --json
```

Joining writes the session's record and a `session_open` line, and signals no
one: the first session learns of a second only by reading the run state, and
never waits on it. Joining again with the same role is a resume; asking for the
other role is refused.

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
claimable again and the lapse is logged. Claiming a record this session already
holds renews the lease. Release it when the lane is done, and leave when the
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
- declares a `--path` in the reading corpus — those lanes are the first's;
- reaches the release step — only the first session cuts a release.

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
`stop`, `refusal`, `pr` and `capture`. For the comparison to count them: a
`lane_close` with `outcome=merged` (or `landed`) is a lane landed, and `backoff`,
`agent_end` and `ceiling_wait` carry `minutes`. The session, window and claim
events belong to their own sub-verbs and are refused here.

## Compare the modes

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement report [--date YYYY-MM-DD | --log <file>] --json
```

Read-only. Per mode: windows, wall clock, lanes opened and landed, the second
session's lanes landed, collisions, lapsed claims, backoffs and the minutes
backed off, agent minutes, ceiling wait and refusals, with each session's share.
`leader` is the mode with the most lanes landed per wall-clock hour — a figure,
not a verdict: the run's own report names the mode it would keep and says why.
Relay any `unparsed` lines; they are counted nowhere.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
