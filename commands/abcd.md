---
name: abcd
description: "Render the status board, or say what one record id is and its next move: Writes nothing; refuses any other positional argument."
argument-hint: "[<record-id>]"
---

# `/abcd` where-am-i

Run the abcd binary's read-only status board for the current repo and present the
result. This command performs **zero writes**.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" --json
```

Then summarise the JSON for the user: the directory, whether it is a git repo,
whether the abcd development record is present, and which `.abcd/` work tiers
exist.

In a repository abcd manages the board also carries one line of presence — the
`statusline` object in the JSON (`state`, `plain`, `elements`), rendered as a
`presence:` line in the text form. It is the same row the host's status line
shows, in plain words: the badge first (`abcd`, `waiting: facilitator` or
`waiting: product thinker`, from the state `/abcd:mode` stores), then the
repository, the branch and the record's counts. Relay the `plain` text when the
state is not `managed`: it says whose answer the loop is waiting on. The field
is omitted in a repository abcd does not manage. The board reads the state and
never changes it; `/abcd:mode` is the writer.

When a sibling worktree or a local branch of this checkout holds a record that
differs here, the board also carries a `peers` object (`live`, `ids`), rendered
as a `peers:` line. Relay it, and point at `/abcd:peers` for the whole picture.
It is omitted when no peer holds anything that differs.

The row itself is produced by `abcd statusline`, the verb the harness runs on
every status refresh with its JSON payload on stdin. In a managed repository it
prints abcd's row; anywhere else it runs the status command that was recorded
at install time and passes its output through unchanged, so the user's own line
is untouched. `/abcd:ahoy install` offers and wires it; nothing here invokes it.

When reports from managed repositories wait in the user account's inbox, the
board carries an `inbox` object (`reports`, `senders`), rendered as an `inbox:`
line. Relay the count and point at `/abcd:inbox`, which lists them; the field is
omitted when nothing waits.

Once a model-tier routing table is accepted — `.abcd/config/oracle-routing.json`
in the repository or `~/.abcd/oracle-routing.json` on the machine — the board
carries an `oracle` array, one object per agent (`agent`, `winner`, `layers`,
each layer with `layer`, `origin`, `tier` and `fan_out`), rendered as an
`oracle:` heading and one line per agent: every layer that holds a row as
`layer=tier`, highest precedence first, the one that applies marked `*`. Relay
the agents whose winning row is not the bundled one. The field is omitted when
no table is accepted, and every delegated step then runs through the harness at
`host-decides`. An orphan row (a name that is no agent) and a fan-out clamped
to the agent's ceiling are reported on stderr; a routing file that cannot be
read omits the lines and says why there.

## Record-id dispatch

Bare answers *what can I do*; `abcd <id>` answers *what is this, and what is
my next move*. A positional matching `^(iss|itd|spc|adr)-[0-9]+$` locates the
record in its store — any status folder or bucket — and renders it read-only:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" <record-id> --json
```

Summarise the `id`, `family`, `status`, `title`, `path`, the `links` edges
(`spec_id`, `intent`, `intents`, `related_intents`, `related_issues`, `resolved_by.*`,
`superseded_by` as present; `intents` is every member a bundle's shared spec lists), and each entry in `next_moves` — the concrete lifecycle move
(e.g. a draft intent points at the planning interview and `intent plan`; an
open issue points at `capture promote` / `resolve` / `wontfix`; decisions are
read). For an issue id the JSON also carries `ledger` — the `checkout` and
`branch` whose ledger was read — because the same id can sit in another
worktree's ledger in another state; name it when you report. A shipped intent's
move reads its fidelity-review marker: an owed review names its receipt and the
re-emit command (`abcd intent audit <itd-N>`); a shipped intent with no marker
owes one too, and the re-emit mints its receipt; a dead-lettered review is
reported unreviewed with its reason; an ingested review leaves nothing to do. A
shape-matching id found in no store exits non-zero naming the stores
searched — unless a peer holds it (a sibling worktree or a local branch, see
`/abcd:peers`), in which case the refusal names that peer's branch, path and
folder instead; relay it, and do not recreate the record here. An issue whose
file is present but was skipped on read exits non-zero naming the file and the
skip reason, which carries the remedy where there is one; relay both, and do
not recreate the record. Any other positional is refused as an unknown command (exit 2) —
there is no `status` alias.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
