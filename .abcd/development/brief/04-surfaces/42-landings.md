# `/abcd:landings` — Say How Each Change Landed

`/abcd:landings` reads how each change reached the default branch
(itd-2610301850454860, spc-2610301950480609): one row per commit on the
default branch's first-parent line, newest first, with the rule that admitted
it. It is a person's verb, listed in the records group beside `status`. It
reads history as this checkout last fetched it, asks the forge about the pull
requests whose answers its cache does not keep, and writes only its forge cache
in the local tier.

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

Bare, the verb lists the rows since the last release tag the tip reaches; a
flag names another start, a tag or a date, and with no release tag and no start
named it refuses at exit 2, naming what to give, as it does when the tag git
describes names no tag the repository holds (an annotated tag whose ref was
renamed is described by the name stored inside it). A failed read of the tags,
or of the ref or commit of the tag named, is not taken for no tag: it exits 1,
as unreadable history does. Three reads are the exception. A lookup of the
default branch that failed is reported as no default branch, at exit 2, because
the shared lookup gives one answer for both. Two answers git gives without
failing are taken as git's answer: a tag ref git's listing skips reads as no
tag, and a tag or tip whose commit object is absent reads as pointing at no
commit. A tag is
looked up as that exact ref, never read as the commit a name in the shape git
describe prints would name. Named one commit,
it traces that commit instead.

## The rows

Each row is classed in order. A person's push is a commit whose subject names
no pull request, or one the forge says did not merge as that commit. A merge is
a person's when a recorded approval of its merged head is found in a worktree
of the checkout, when the forge holds a review approving that head by someone
other than the author, or when the first parent committed the human mode. Under
the auto mode, a merge with no approval seen is the drain's when its branch
carries the drain's admission trailer, and automatic otherwise. A row the
evidence cannot place with a person is marked unattended, and a row whose pull
request the forge's answer was not read for is unattended in every mode.

The text says what the machine form carries. Each row's first line holds the
commit, the pull request, the class, the rule or the committer name git holds
for a person's push, the lane kind, the approval, where it was read and whether
one was seen, up to eight of the issues and intents the row resolves and
delivers, its mode, the unattended mark, and the commit that reverts it where
one does; the next says when it landed, the head it merged, where its rule was
read and up to eight of the values saying who assisted. Under them come, never
dropped and each cut at 600 bytes, what the forge said against the pull request
the subject names and why the forge's answer was not read; then the revert lines
the row's commits carry, each with the row it marks or why it marks none: those
that mark no row first, the untraceable ones before the rest, up to sixteen,
then up to eight that mark a row, then how many more of each and how many of the
hidden ones that mark no row are untraceable. A row can carry many lines that mark no row,
one for each of its commits that reads as git's revert and names nothing it
resolves and up to 64 for each commit's message or pull request body, so they
have a cap of their own, and no line that marks a row is printed while one that
marks none is held back. Then come the notes (how the range was found, why
the forge was not read for some row, why an answer was not kept, why a first
parent's mode could not be read) and the tip read. The machine form prints the
reader's listing with the same rows and stable keys. A filter keeps the
unattended rows alone, saying so when none is, and another lists the reverted
rows grouped by rule, each group with a count, and the rows carrying an
untraceable revert in a group of their own; the two are not taken together. A
listing whose forge answers were not read still exits 0, its rows unattended
and the forge note saying how many.

## The trace

Named one commit, the verb prints the row that brought it onto the default
branch however far back, with its class, rule and revert lines, and for a
reverting commit the rows it reverts, with notes saying what its range could
not place. A commit the tip does not reach is said to be not on the
default branch, at exit 0; a name that names no one commit is refused at exit
2, and so is a trace given a listing's filter or start.

## What it prints of typed text

The committer name, the forge's logins and failure text, the lane, the ids, a
revert line's names and the notes carry text someone typed. The reader masks
each for a terminal and bounds it where it reads it, and the render masks every
value it prints again, cuts each field at 200 bytes and each note at 600, and
caps a row's lists as the rows section says, each then saying how many more, so
one row's text is bounded whatever the reader hands it. The machine form writes the runes the terminal mask masks as
its own escapes. The number of rows is bounded only by the history the reader
takes, whose log it refuses whole past 64 MiB.

## What it touches

It fetches nothing and pushes nothing. Its one network act is the reader's:
it asks the forge client about each pull request the range names whose answer
its cache does not keep, up to twenty to a document. The cache keeps only the
answers that cannot change: one accepted for its row, one saying the pull
request merged as another commit, and the forge's word that a number names no
pull request. An answer not read, one whose reviews run past the page read, one
read without the commits its row asked, and one saying the pull request has not
merged or naming no merge commit are not kept, so every listing asks for them
again. A document whose answer is too large, or carries an error naming no one
pull request, is asked again in halves, within a bound on the calls a listing
spends so. A squashed revert's pull request body is asked apart, on every
listing that reads it, and never kept.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd landings`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--reverted` | bool |
| `--since` | string |
| `--unattended` | bool |

<!-- surface-appendix:end -->
