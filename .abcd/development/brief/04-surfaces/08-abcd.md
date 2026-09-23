# `/abcd` — Top-Level Where-Am-I Status Board

Type one word and see where you left off. The bare top-level command is the
cross-verb re-orientation surface: not "what does `capture` think", but "what is
the state of this project right now, and what should I do next". It is strictly
read-only, so it is safe to type at any moment, including the moment you are
least sure what is safe.

It complements the per-verb bare renders (`/abcd:ahoy`, `/abcd:capture`, and the
rest), each of which is scoped to its own surface. This one is the cross-verb
answer.

## What ships today

Two read-only forms, and no third.

**Bare `abcd`** renders a four-field snapshot of the current directory: the
directory itself, whether it is a git repo, whether an abcd record is present,
and which of the `.abcd/` work tiers exist. The plugin command invokes its JSON
form.

**`abcd <record-id>`** takes a single positional matching `iss-N`, `itd-N`,
`spc-N` or `adr-N` and reports, read-only, what that record is, where it lives,
and the concrete next move for its lifecycle state. Bare answers *what can I
do*; the id form answers *what is this, and what is my next move* (spc-26,
itd-121). A positional on the namespace root is not a `show` sub-verb, so the
form stays inside the naming discipline.

Any other positional is refused: the CLI exits **2** with `abcd: unknown command
…` on stderr, which is the framework's usage-error convention. `abcd status` is
refused that way, and `abcd help` prints the framework's usage text and exits 0.
A shape-matching record id found in no store is a structural fault: the command
exits non-zero with a diagnostic naming the store it searched, never a silent
fall-through to the snapshot.

Binary-backed `/abcd:` verbs route through the transport-agnostic core (the CLI
is the front door today; an MCP server follows later, per
[adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)). Not every verb
does: `consult` and `ingest` run entirely as host-side markdown over the
sources corpus and never invoke the binary. `prepare-this-repo` is the mixed
case: its audit half runs `abcd lint`, and its adoption half is binary-backed
too and writes — the identity verb's initialiser records the repo's identity
block and registers the surfaces held to it, and the ahoy installer lays the hooks, the

banlist stub and the gitignore rules. What the markdown owns is the interview
around them: which file carries the identity, what the tagline should say,
whether the attribution gate is wanted. The command decides; the binary writes.

**The presence line** (itd-200, spc-70) is the one addition the shipped board
has taken since: in a repository abcd manages, the text render carries a
`presence:` line and the JSON a `statusline` object, both the plain form of the
same row the host's status line shows — the badge first (`abcd`, `waiting:
facilitator`, `waiting: product thinker`), then the repository, the branch and
the record's counts. The state behind the badge is what `abcd mode` stores at
`.abcd/.work.local/mode`; the board reads it and never writes it. In an
unmanaged repository the line is absent and the field omitted. The board is the
fallback for a host with no status surface, so it renders the line even where
the user-level setting has switched the status line off, and it never runs the
previous status command that `abcd statusline` falls back to.

## The board itself is not built

> **Design target (itd-20, `intents/planned/`, `spec_id: null`).** Everything in
> the rest of this chapter describes a board that does not exist on any shipped
> surface. The shipped status path reads `.git`, `.abcd/development` and the
> three work tiers, and nothing else: no visibility, no disembark log, no
> dev-sync record, no logbook, no linked intents, no spec store, and no
> thresholds, walks or timeouts. There is no alias routing either: `status` and
> `help` route to no render.

The board renders exactly six sections, always in this order, each source a
local-filesystem read, and each source's absence mapping to a named known-state
line rather than an exception or a silent omission.

1. **Project and visibility**: the project name and the repo's declared
   visibility.
2. **Last disembark**: when this repo was last disembarked and to where, as an
   age in days with a staleness flag past seven days. A week-old rescue snapshot
   warrants a re-pack cue without nagging on daily work.
3. **Dev-sync staleness**: permanently a known-state line, for the reason below.
4. **Recent logbook**: the last five entries by modification time, the render
   descending exactly one bounded level into each category directory. Two levels
   total, never a recursive walk. Fewer than five renders what exists with a
   count.
5. **Active intents**: intents carrying a linked spec that is not done, with the
   spec status read from the native spec store.
6. **Suggested next actions**: a short bullet list keyed off the state above.

Each absent source names itself: no visibility recorded, no disembark yet with
the verb to run, no dev-sync record, no logbook entries yet, no intents with a
linked spec, an unknown spec status. Outside an abcd repo entirely, a single
guidance line replaces the whole board.

The bounds are part of the design, because a re-orientation command that is slow
is a command nobody types: bounded directory reads only, last-N sorting rather
than full-history loads (logbook capped at five, active intents at ten), and a
five-second timeout on the spec-store read whose expiry maps to the unknown
status line rather than wedging the render.

**Zero writes** is the guarantee, and proving it is part of the delivery: a
static zero-mutation check over the render module, and a filesystem-snapshot
test asserting that a render over a populated fixture repo mutates nothing at
run time. Neither exists yet; the shipped status path has its own tests over
temporary directories.

## Two open questions the design has to answer

**Where the last-disembark section reads from.**
[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md) removed
this section's only source: there is no in-tree lifeboat directory to stat,
because disembark is read-only and writes to an operator-chosen destination,
with the record of what was disembarked living at the operator level under
`~/.abcd/voyage/`. The natural replacement is to read that log, keyed on this
repo's root-commit SHA. That is unsettled, and three things have to be decided:
whether the board reads outside the repo at all, when every other section is a
local read; whether showing an absolute destination path on screen is
acceptable, given the same privacy concern that moved the voyage log out of the
tree; and, if neither, whether the section is simply dropped, leaving five.

**Dev-sync staleness has no substrate, and may never.** The `dev-sync work`
migration surface is itself a design target: no `dev-sync` verb is on any
shipped surface, and no dev-sync code exists in the tree. Even in that design it
is migration logic rather than a durable last-run timestamp, and no config field
or store record captures when it last ran. So section 3 is permanently its
known-state line until a state substrate exists. That is an allowed terminal
state, recorded here and in itd-20 so it is not read as a shipped capability.

## On `status` and `help` as aliases

The naming discipline forbids a sub-verb that names what bare already renders.
`status` and `help` are not that: as aliases they would route to the identical
bare render and add no behaviour, existing only so the intuitive spelling lands
on the same board. They are therefore admissible under the rule rather than
exceptions to it. Had they introduced divergent behaviour, they would be
forbidden. Nothing routes them today.

## Related documentation

- Naming discipline: [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)
- Intent: [`itd-20`](../../intents/superseded/itd-20-top-level-abcd-dispatcher.md)
- The command surface this board sits at the head of: [`README.md`](README.md)
- The per-verb bare renders it complements: [`05-intent.md`](05-intent.md), [`01-ahoy.md`](01-ahoy.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd`

| Flag | Type |
|---|---|
| `--json` | bool |
| `--no-color` | bool |

### `abcd mode`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
