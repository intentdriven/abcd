---
name: mode
description: "Print or set whose answer the agent loop is waiting on: Writes the state only when setting it; refuses an unknown state or a checkout with no local tier."
argument-hint: "[managed|facilitator|product-thinker|autonomous|back|queue]"
block: agents
user-invocable: false
---

# `/abcd:mode`

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

## Set the state — two writers, one verb

**The agent, at a question.** A question through the host's question tool
needs no setting: its chip already names whom it is for. When the guard admits
abcd's question in a repository abcd manages, it sets the state from the chip's
role (`Product` names the product thinker; `Tech` and `Setup` name the
technical facilitator), so a chipped question is never refused on the mode,
and it marks the question open. The answer resets the state to `managed`: the
plugin's hook on the question tool's result resets it when the answer comes
back, and on a host that does not run that hook the prompt hook resets it on
the next human message. Either says so once on stderr, so you do not set it
back yourself. While the away mode below holds, the guard refuses every
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

While it holds, an act only the person may approve is queued for their
return instead of asked, and you carry on with other work:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode queue "<step>" --json
```

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
the path of a build of the latest reviewed default branch (`origin`'s, as last
fetched) once it is ready: run that path in place of the plugin-root binary,
and the plugin-root binary while it prints nothing.

**User input:** $ARGUMENTS
