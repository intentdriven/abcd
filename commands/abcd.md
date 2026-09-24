---
name: abcd
description: Top-level where-am-i status board and record-id dispatch. Bare `/abcd` renders a read-only snapshot of the current directory; `/abcd <record-id>` (iss-N, itd-N, spc-N, adr-N) reports what that record is and the next move. Strictly read-only.
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

## Record-id dispatch

Bare answers *what can I do*; `abcd <id>` answers *what is this, and what is
my next move*. A positional matching `^(iss|itd|spc|adr)-[0-9]+$` locates the
record in its store — any status folder or bucket — and renders it read-only:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" <record-id> --json
```

Summarise the `id`, `family`, `status`, `title`, `path`, the `links` edges
(`spec_id`, `intent`, `promoted_to`, `resolved_by.*`, `superseded_by` as
present), and each entry in `next_moves` — the concrete lifecycle move
(e.g. a draft intent points at the planning interview and `intent plan`; an
open issue points at `capture promote` / `resolve` / `wontfix`; decisions are
read). A shape-matching id found in no store exits non-zero naming the stores
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
