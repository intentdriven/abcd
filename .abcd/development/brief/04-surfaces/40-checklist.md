# `/abcd:checklist` — Keep the Working Copy's Checklist on Disk

`/abcd:checklist` keeps what a session in this working copy is doing on disk
(itd-2610170638184486, spc-2610230805429429 steps 1 to 6): what it works on,
what it waits on and what it has promised. A resumed or a fresh session in the
same copy reads the same list, and every session start, a compaction's
included, adds it back to the session's context. Two of the spec's steps have
not run: the crash lab (step 7), so nothing here yet shows the list surviving a
crash or a power cut, and the live capture in a real session (step 8), so the
re-injection after a compaction is shown by the hook's tests, not yet in a live
session.
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

The reader takes only a file this working copy's own write could have made:
it refuses a file git tracks, staged or committed, since that came with the
checkout; a file group or others can read, or another account owns, since the
write makes it 0600 as the person; a file with a second name (a hard link),
since the write makes it by a rename; a file under a local tier that is not a
plain directory (a symlink, or a submodule's top, whose files the outer git
does not list); and, inside a git checkout, every file when git cannot say
whether they are tracked. The file it opens is checked the same way, not only
its path, and must be the file the path named. Otherwise it validates and
scans the file. When the
file fails, the reader judges the last good copy the same way and says why it
fell back; when both fail, it reads nothing and says so. A reason never
quotes the file. A change is held to the same gate, so no item is written
that the next read would drop: while git tracks the list it refuses with exit
2 and says to untrack it with `git rm --cached`; it refuses a tier the reader
refuses; and it refuses with exit 2 a list the reader refuses for its mode or
owner, and a working copy whose filesystem does not keep the owner-only mode
or the owner, which a write shows.

The guard refuses the host's own file tools any path ending in the local
tier's two checklist files, wherever it is, in a working copy or not, whatever
case the path is spelled in, and names the verbs to use instead. A path
beginning with `~` or `~/` is judged both as written and with the home
directory in place of the tilde, as a shell expands it. It reads each
path as written, cleaned and then through the links of the folders on it that
exist, and as the volume resolves it; when a reading ends in a link, it reads
that link's target the same ways, and so on down the chain. It refuses outright
a path it cannot finish reading: a path, a link target or any reading of them
longer than a fixed 4096-byte limit, a chain of more than 40 links, a path
whose readings would hand the link lookups too many path parts, and a path the
guard cannot read, because a lookup fails for any reason but a part that does
not exist yet: a name too long for the system, a loop of links in a folder on
the path, a folder the guard may not search. It also refuses a path whose
readings take longer than twenty seconds, which only a slow or busy machine
reaches, and that refusal says to try the call again.
That refusal only catches a mistake: a shell line still reaches the file, and the
reader is what holds, since a file written past the verbs is judged like any
other and a failing one gives way to the last good copy.

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
list is given nothing, and the list never fails a start. The list's part of
the start runs within the start's own deadline of one minute, counted from the
hook's start, every git call it makes included: the checkout's root, the
read's tracked check and the secret scanner's reading of the person's git
identity. A deadline spent before or during it runs no further git and adds
nothing; where the checkout holds abcd's folder it says so in one line on the
hook's error stream, and the start goes on.

## At every compaction

No hook can make a session bring its list up to date before the host compacts
its context: the hook that runs then may run a command or block the
compaction, and nothing more. So the list is kept current as the session
works, the next start adds it back, and the compaction itself is recorded.
Each one appends a line to `.abcd/.work.local/checklist.events.jsonl` under
the schema `abcd/checklist-events/v1`: the time and the trigger, `manual` for
a compaction the person asked for, `auto` for one a full context set off, and
`other` for any trigger the host does not document, so nothing else the host
sends is copied in. The file is made 0600 in a plain local tier; a file at
that path that is not a plain file this account owns, with one name and
readable by it alone, is left as it is and not added to, and the file opened
for the line is checked the same way, not only its path. The record
never blocks: a compaction that could not be recorded still goes ahead, and the
hook says so in one line on its error stream. Outside a managed checkout, and
on a drive's walled stage, nothing is recorded.

## When the list is behind

The list is behind when the work the session owns holds something no item
names, or moved after the list was last written. The stop check reads that
work as it reads owed work: each branch of the session's own, each live
reservation under this working copy's key that no other live session's run
holds, and, while the person is away and the forge is read, each own
branch's open pull request, which an item may name in place of its branch.
An item names work in its owner_refs; a branch whose head was committed after
the list's updated_at has moved.

While the person is away, a turn's end with the list behind is blocked once
per consecutive set of references, naming them: on a stop the away check already
blocks, the note is added to its reason, and on one it allows, the note is a
block of its own, counted toward the host's cap on consecutive continuations
as every block is and never past it. Only the set last blocked on is kept, so
the same set does not block again on the stops that follow it, while a
different set blocks, and so does a set that comes back after another; the
cap bounds the blocks either way. While the person is present, the stop is never
blocked for it, and the stop check reads disk alone, calling no forge; a
stop's message reaches the person, not the model, so the note is kept in the
session's stop state and the next prompt carries it into the model's context,
once, under a CHECKLIST heading, and a set is noted once per consecutive
set, on the same terms. The stop check runs every git call of its read within
its one deadline but one: the reservations' view, which reads this copy's
reservations and is not yet bound to it (iss-2611100548341325). The away
session's state, read once that deadline may be spent, runs under a
two-second bound of its own. How a spent deadline shows depends on where it is
spent. When the reading of
owned work is cut short, the list is not read: the stop goes through with that
reading's own message to the person, saying what was not read, adds no block
for the list, leaves no note and writes no line about the list. Only a
deadline spent at the list's own read writes one line on the hook's error
stream; the stop then goes as judged without the list, adds no block of its
own and leaves no note. A deadline spent before the checkout is named allows
the stop, with one line on the error stream saying so where the checkout holds
abcd's folder, and nothing anywhere else.
A step the session never wrote down and that left no branch or reservation
behind cannot be seen.

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
