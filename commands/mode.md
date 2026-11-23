---
name: mode
description: "Print or set whose answer the agent loop is waiting on: Writes the state only when setting it; refuses an unknown state or a checkout with no local tier."
argument-hint: "[managed|facilitator|product-thinker|autonomous|back|queue]"
block: agents
user-invocable: false
---

# `/abcd:mode`

<!-- generated: ask-the-binary -->
Before you fill in an input for this command, run `"${CLAUDE_PLUGIN_ROOT}/abcd" explain mode`, and where its answer and this page disagree, follow the binary, since this page may be older than it.
<!-- /generated -->

The waiting-on state behind the status line's badge. It answers one question
for everyone at once: is abcd here and nobody waiting (`managed`), is the loop
parked on the **facilitator** — the person at the terminal running the agents
— or is it parked on the **product thinker**, who answers on a surface of
their own and is exactly the person a terminal-only stop leaves unnotified.
The status line, the bare `/abcd` board and the agent all read the same stored
state, so no surface invents its own answer.

## Print the state

Bare invocation is read-only. Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode --json
```

Then tell the user the `state`. An absent store reads as `managed`; outside any git checkout the print form refuses (exit 2), because there is no repository whose state could be read.

While an away record holds (below), the JSON also carries `away`: `until`,
the time the away mode ends, `open_ended`, and `queued`, how many steps wait
for the person's return; the plain form prints them on a second line, as
`away until 2026-10-19 09:00 BST, 2 queued`. Tell the user both. With no away
record holding, `away` is absent and the plain form prints the state alone.

## Set the state — two writers, one verb

**The agent, at a question.** abcd's question is headed with its number
(`Q1`, `Q1a`, `Q2/5`), which names no person, so set the state to the person
the question is for before the question is written, so the check the
`abcd:question-drafter` agent runs on each draft reads that person, and keep
it while you ask:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode product-thinker --json
```

or `facilitator` for the technical facilitator. The status line then names
whom the question on screen is for, and the guard holds the question to that
person's register: while the state names the product thinker, a question that
names a record number or a command is refused. When the guard admits the
question in a repository abcd manages, it leaves the state as you set it and
marks the question open. A question headed in the older form, a role word
first (`Product Q2`), still sets the state from its role (`Product` names the
product thinker; `Tech` and `Setup` name the technical facilitator). Either
way abcd's question is never refused on the mode. A question whose header
misses abcd's grammar (`Q 1`, `q1`, or none) is not checked, but it is still
marked open under the state you set, so the answer resets it, and so is a
question the guard refuses, so a refused question you then abandon does not
leave the state parked. The answer resets the state to `managed`: the
plugin's hook on the question tool's result resets it when the answer comes
back, and on a host that does not run that hook the prompt hook resets it on
the next human message. Either says so once on stderr, so you do not set it
back yourself. The reset applies only while the state is still the one the
question was asked under: a state set again since, by this verb or by hand,
is left as set, and setting a state with this verb clears a question still
marked open under another state, keeping one marked under the state it sets,
which the answer then still resets. A question you abandon after the question writer refused it
reached nobody, so set the state back yourself with `abcd mode managed`. While the away mode below holds, the guard refuses every
question instead, whatever its chip, and sets nothing: its reason says which
acts to queue and that every other question is decided with `abcd decide`.

**The agent, at a stop that is not a question.** When you stop to obtain a
verdict any other way, such as a hand-off or a step the person runs, record
whom you are addressing *before* you stop, so the parked stop is visible on
the status line and the board while you wait:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode product-thinker --json
```

Use `facilitator` when the verdict is the technical facilitator's to give, and
set it back when the loop resumes:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode managed --json
```

**The human, by hand.** `/abcd:mode facilitator` or `/abcd:mode product-thinker`
says which hat the person wears; the agent's next read returns it. Pass the
user's word through unchanged.

The vocabulary is closed: any other word is refused (exit 2) with the three
named, and nothing is written. The state lives per checkout at
`.abcd/.work.local/mode`, which only a repository abcd manages has, so outside
one the set form refuses (exit 2) and creates nothing — say so, and point at
`/abcd:ahoy install` if the user wanted this repository managed.

## The away mode — `autonomous`, `queue`, `back`

The person's word that they are away and sessions should work on until they
are back. It sits beside the badge and is not a fourth word. One record holds
it for every working copy of the repository on this machine, in
`~/.abcd.noindex/away/<root-sha>/record.json`, changed under its lock, which
only `back` goes around when it cannot have it.

**The away mode grants no rights.** The person's own standing instruction
stays what decides whether a session commits, pushes, or opens or merges a
pull request, exactly as before. Never read the away mode as permission.

Set it when the person says they are away — they run it themselves, or you
run it on their word in this session:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode autonomous --until 09:00 --json
```

`--until` takes a clock time (its next occurrence, local time) or a timestamp
(`2026-10-19T09:00`, or RFC 3339). Without it the mode is open-ended; every
record ends seven days after it was set at most, and an `--until` past that,
or one already past, is refused (exit 2) with nothing written. Add
`--hold-pushes` only when the person says pushes wait for their return: it
narrows what sessions do and grants nothing. Tell the user the `terms` the
verb prints. A prompt never ends the mode.

**A session whose setting reads as off a profile is warned.** After the set, one
line on stderr opens `WARNING:` (stdout's JSON is untouched) when one setting in
this session's environment reads as off a profile. A host that reports no such
setting reads as off, and every doubt falls towards the line. Pass the line on to
the person, as it asks. The mode is set and the exit code is the same either way:
this warns and never refuses. This reads one setting and proves no wall, and the setting's value changes nothing
the session holds. No abcd command starts
a session on a profile yet. A profile is a separate login and settings on one
operating-system account, and a stage's shell reads what the account can.
`abcd drain --drive` is the unattended start: it runs agents of its own on a separate
login, with named tools, checked at its start.

While it holds, an act only the person may approve is queued for their
return instead of asked, and you carry on with other work:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode queue "<step>" --json
```

To read what is queued without ending the mode, run it bare:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode queue --json
```

It lists the steps queued in a record not yet ended, oldest first (one per
line in the plain form; `queued` and `steps` in the JSON), and writes
nothing: it takes no lock, and the away mode holds as before. With no record,
or one already ended, it lists nothing and exits 0.

A queue with no away record holding is refused (exit 2): ask the person
instead, the product thinker or the technical facilitator, whichever the
question is for. While the mode holds, `mode product-thinker` and `mode facilitator`
are refused too, naming `abcd decide`: decide the question and record it, or
queue the act.

End it only when the person who was away, the product thinker or the
technical facilitator, says in this session that they are back:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode back
```

It prints one plain line per queued step, per item handed back and per
recorded stop (a stop at the host's cap is labelled "possibly forced by the
cap"), records who ended it and from which working copy, and closes the
record; on a record whose time has passed it prints the same and closes it.
Relay every line to the user. A record or lock that is not a regular file this
account owns that no one else can write, or a record that does not parse, is
refused by `autonomous` and `queue` (exit 2), naming the file; so is one past
64 KiB, except that `autonomous` carries a record merely past 64 KiB, letting
its oldest lines go. `back` ends the mode
whatever state the record is in: one past 64 KiB is printed whole and kept
with its oldest lines let go and counted; one that does not parse, or is not
safely this account's, is replaced by a closed record, printing nothing from
it; a lock not had in time is gone around. Each is named on its own line:
relay those too. A queue that is full is refused (exit 2) and never cut: say
so, and carry on with other work.

Once the away time has passed with a step or stop unread and the record not
yet closed, every prompt gives you one line saying the window ended and that
its checks no longer hold: questions and turn ends are judged as when the
person is present. Do not set the mode again on that line; set it only on the
person's word in this session. The person's own line says the same and points
at `back`.

## Where the host has no status surface

The set form prints one line naming whose answer is owed — for example `abcd:
waiting on the product thinker — an answer is owed` — when this machine has no
status line installed (no `~/.abcd.noindex/statusline.json`, or `disabled` set in it).
It prints once, because the verb call is the stop; setting `managed` owes
nobody and prints nothing, and with a surface installed nothing is printed.
With `--json` the line arrives as the `notice` field: relay it to the user
verbatim when present. The bare `/abcd` board shows the same state on demand.

The verb performs no network request in either form.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

<!-- generated: command-facts -->
**Command facts, format version 2.** Where this block and `"${CLAUDE_PLUGIN_ROOT}/abcd" explain mode` from the running binary disagree, follow the binary.
<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->

- `abcd mode [<state>] [flags]`
  - An example call is `abcd mode`.
- `abcd mode autonomous [flags]`
  - An example call is `abcd mode autonomous --until 09:00`.
- `abcd mode back [flags]`
  - An example call is `abcd mode back`.
- `abcd mode queue [<step>] [flags]`
  - An example call is `abcd mode queue "approve the embargo close for the example advisory"`.
<!-- /generated -->

**User input:** $ARGUMENTS
