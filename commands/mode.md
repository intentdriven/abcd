---
name: mode
description: Print or set whose answer the agent loop is waiting on — managed, facilitator, or product-thinker — by invoking the abcd binary. The bare form is a read-only print; the set form writes one line to the checkout's local tier.
argument-hint: "[managed|facilitator|product-thinker]"
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

Then tell the user the `state`. An absent store reads as `managed`.

## Set the state — two writers, one verb

**The agent, at a stop.** When you stop to obtain a verdict, record whom you
are addressing *before* you stop, so the parked stop is visible on the status
line and the board while you wait:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" mode product-thinker --json
```

Use `facilitator` when the verdict is the facilitator's to give. When the
answer arrives and the loop resumes, set it back:

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

## Where the host has no status surface

The set form prints one line naming whose answer is owed — for example `abcd:
waiting on the product thinker — an answer is owed` — when this machine has no
status line installed (no `~/.abcd/statusline.json`, or `disabled` set in it).
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
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
