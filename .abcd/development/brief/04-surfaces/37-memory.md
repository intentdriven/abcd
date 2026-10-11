# `/abcd:memory` — Keep Memory Notes

`/abcd:memory` keeps memory notes: short lessons an agent writes so that a later
session does not learn them again (itd-2610091918433290,
spc-2610111627213138). It is an agents' command: its page is hidden from a
person's menu, and the binary's help lists the verb with the agents. Recall of a note through the prompt hook is the same
spec's later step; this chapter covers the store and the verb.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only._

| Verb | Bucket | Status |
|---|---|---|
| `add` | — | shipped |
| `list` | — | shipped |
| `remove` | — | shipped |

## Two layers

A `memory.json` sits beside each `rules.json` layer. The machine's
`~/.abcd.noindex/memory.json` holds the person's own notes; a repository's
`.abcd/memory.json` holds the notes a person promoted into the project. Both
are read with the checks the two rules layers get: the machine file only as a
regular file this account owns that no one else can write, within 256 KiB and
never behind a symlinked `~/.abcd.noindex`; the project's file through the
repository layer's guarded read, from the root the rules loader resolves. A
file that fails a check or does not parse is named with its reason, in tilde
form, and never silently skipped. An absent file costs nothing and nothing is
created by a read.

Each note carries an id minted in the `mem` family, a one-line title of at
most 120 characters, a body of at most 2 KiB, one to twelve recall words or
short phrases, and its stamps: who wrote it (the `Assisted-by:` form
`<Vendor>:<model-version>`), when (the clock's UTC time, never the caller's),
in which session (the host's own id, or null where the host supplies none,
shown as "session unknown"), and its origin (`agent`, or `host-memory-move`
for a lesson moved by hand from the host's own memory files). A personal note
is tied to the repository it was written in by that repository's root commit,
or carries a null root and is recalled in every repository. The project's file
carries no root.

## Writing

Agents write only the machine layer, and only through one gate, before
anything is written:

- the shared scanner, through the memory-page writer's own redactor, refuses
  on a degraded scanner and on any span the page writer would have to rewrite
  (a secret, a token, an identity or network span, the caller's home path) in
  any field;
- the private-names banlist refuses a name this checkout's private layer holds
  and, in a linked worktree, the primary checkout's; a store that cannot be
  read counts as a match.

The refusal names the field and the kind of span, never the span, and the
agent rewrites the note in its own words: the gate never redacts and writes.
A session id shaped like a link is refused, so no session URL enters a note. A
note written to be recalled everywhere has been judged against the writing
checkout's private names only, since there is no machine-wide banlist layer,
and the write says so. A write with no session id says, on that write, that
the session is recorded as unknown.

A write creates the abcd home if it is missing (refusing a symlinked one),
takes an exclusive lock on that directory, re-reads the file under the lock,
and replaces it atomically at mode 0600, so two agents writing at once both
land. A machine file the read refuses is never overwritten.

Adding takes no path: the text arrives on stdin, so the verb never opens the
host's memory files, on this verb or any other. The page tells the agent to
move a lesson from them only at the person's word and never to edit or delete
the original.

## Reviewing

Until the project's review on the dashboard ships (itd-2610111655362931),
listing and removing are the review. Listing shows every note in both layers
with its body, recall words, stamps and reach (this repository, everywhere,
another repository, or the project's file). Removing takes any personal note
out of the machine layer, whoever wrote it, run by a person or by an agent at
a person's request. A note in the project's file is refused, naming the file:
it is removed by editing that file in a reviewed change. There is no promote
operation.

## Exit codes

`0` done; `1` an I/O failure or a machine file the read refuses, with nothing
written; `2` refused (a field past its limit, the gate, no root commit for a
tied note, an absent note, a note in the project's file), with nothing written.
The JSON output holds on every path: a refusal is the `{"abcd":"error",…}`
envelope on stdout.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd memory`

Sub-verbs: `abcd memory add`, `abcd memory list`, `abcd memory remove`.

Flags: none.

### `abcd memory add`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--by` | string |
| `--everywhere` | bool |
| `--moved-from-host` | bool |
| `--recall` | stringArray |
| `--session` | string |
| `--title` | string |

### `abcd memory list`

Sub-verbs: none.

Flags: none.

### `abcd memory remove`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
