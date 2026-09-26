---
name: build
description: "Start the loop that takes one READY intent to delivered: Writes the run's state file in the local tier; refuses an open question, a hold or a peer holding it."
argument-hint: "<itd-N>"
block: people
---

# `/abcd:build`

Take one intent from READY towards delivered with the loop holding the run, not
this conversation. The run lives in a state file; every invocation reads it,
does at most one step, writes it and exits, so a session that stops, is
compacted or is killed loses nothing, and the next invocation resumes where the
last one stopped.

## Start the run

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" build <itd-N> --json
```

For an intent with no run in progress, the checks run first, and every one must
pass:

- `key` — the argument is an intent id. An issue id is refused: the issue key
  is not built yet.
- `ready` — the intent is READY: planned, its criteria written, its spec linked
  and written (the same gate `/abcd:intent` reports).
- `open_questions` — no open question under `## Open Questions`: every list
  item there counts unless the section opens with an italic `_All resolved …_`
  line or the item is explicitly marked resolved or deferred (`**Deferred**`,
  `resolved:`, `**explicitly deferred**`).
- `claim_sections` — the `## Mechanism` prompt is answered (or the section
  absent) and the scope conditions are recorded.
- `hold` — the intent carries no `held:`.
- `steps` — the spec's `## Steps` reads, and at least one step is not landed.
- `peers` — no peer holds the intent: no sibling worktree or local branch holds
  it in another bucket, and no session holds a live claim on it. A peer that
  cannot be read (a worktree git will not answer for, a ledger holding one id
  twice) and an unreadable claim count as holding it: what they hold is unknown.

A refusal writes nothing. It exits 2, or 3 when a peer holds the intent (back
off and take other work). Under `--json` the refusal comes as its own document
before the error envelope: `refusal.step`, `refusal.check`, `refusal.reason`,
`refusal.remedy` and every check's row in `refusal.checks`. Tell the user the
check, the reason and the remedy, and do not work around it: an open question
goes back to the planning interview, a hold to the person who placed it.

When the checks pass, the payload names the `run_id`, the `state` file
(`.abcd/.work.local/run/<run-id>/state.json`), the first `lane` (the spec's first
unlanded step), the `pending` spec steps, and `next`, the move to make. The
local tier is never created: in a repository abcd does not manage the verb
refuses. Starting again while the run is in progress creates nothing, runs no
check, and reports `resumed: true` with the same run and an empty `checks`: the
run's own lanes move and claim the intent, so judging it again would refuse the
run as its own peer.

## Drive it

The host session drives the loop. Take one step at a time:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement step --json
```

`performed` names the step the call completed. When a step hands work to an
agent, `awaiting` names the `role` to start as a fresh agent, the `brief` to
hand it and the `receipt` path it writes to. Start that agent, and when it
returns hand the receipt back:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement receipt <path> --json
```

The lane advances only on a receipt that verifies. Asking for a step while the
lane awaits a receipt re-tells what it awaits and moves nothing.
`"${CLAUDE_PLUGIN_ROOT}/abcd" implement status --json` renders every run, its
lanes and its record, and writes nothing.

In this build the lane's steps are named but their bodies are not carried yet:
the first step is refused naming the spec piece that delivers it, and the run
stays as it is, ready to resume in an abcd that carries it. Report that refusal
as it is; do not make the worktree, the brief or the pull request by hand on the
run's behalf.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
