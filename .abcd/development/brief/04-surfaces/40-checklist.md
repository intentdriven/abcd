# `/abcd:checklist` — Keep the Working Copy's Checklist on Disk

`/abcd:checklist` keeps what a session in this working copy is doing on disk
(itd-2610170638184486, spc-2610230805429429 steps 1 to 3): what it works on,
what it waits on and what it has promised. A resumed or a fresh session in the
same copy reads the same list, and the list survives a compaction or a crash.
It is an agent's verb, listed in the agents group, and every sub-verb runs in
the process, touching no network.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `add` | — | shipped |
| `set` | — | shipped |
| `show` | — | shipped |
| `render` | — | shipped |

The bare verb lists every item and field and writes nothing. Adding an item
mints a stable short id made from its title; setting changes an item's state;
showing shows one item; rendering prints the list as one block of text,
open items first.

## The file

The list is one JSON file per working copy, `.abcd/.work.local/checklist.json`,
under the schema `abcd/checklist/v1`. Each item carries an id, a title of at
most 200 characters, a state (working, waiting, done or dropped), what it waits
on, when to look again, a promise, the references it is about, and an optional
credential named by kind, scope and expiry only (brief invariant 24,
[adr-2610222026265719](../../decisions/adrs/2610222026265719-a-session-s-checklist-names-a-credential.md)).

## The write and the read

Every write validates the list and scans each field with the native secret
scanner before anything reaches the disk. A refusal names the field and the
rule or pattern kind, never the value, and exits 2. Every change reads,
changes and writes the list under one lock, so two writers in the same copy,
a session and its sub-agents among them, never lose each other's items. The
file is written atomically, and the same checked bytes are kept beside it as
the last good copy, so a file changed by anything other than the verb never
becomes the last good copy.

The reader validates and scans the file. When it fails, the reader validates
and scans the last good copy and says why it fell back; when both fail, it
reads nothing and says so. A reason never quotes the file.

## At every session start

Every session start, whether a startup, a resume, a clear or a compaction,
adds the list's rendering to the session's context, so what the session wrote
down outlives the compaction that drops it from memory. The rendering shares
the host's cap of 10,000 characters on the start's whole output with the away
block and the update notices the same start prints: those stay whole, and the
list gets what they leave, less a margin of 500. Past that, it keeps the open
items, newest first, and says how many it left out and how to see them all.
When the file fails the reader, the last good copy is added with one line
saying why; when both fail, that line alone. A working copy that never had a
list is given nothing, and the list never fails a start.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd checklist`

Sub-verbs: `abcd checklist add`, `abcd checklist render`, `abcd checklist set`, `abcd checklist show`.

Flags: none.

### `abcd checklist add`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--next-look` | string |
| `--promise` | string |
| `--waits-on` | string |

### `abcd checklist render`

Sub-verbs: none.

Flags: none.

### `abcd checklist set`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--state` | string |

### `abcd checklist show`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
