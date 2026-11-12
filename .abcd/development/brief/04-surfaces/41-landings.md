# `/abcd:landings` — Say How Each Change Landed

`/abcd:landings` reads how each change reached the default branch
(itd-2610301850454860, spc-2610301950480609): one row per commit on the
default branch's first-parent line, newest first, with the rule that admitted
it. It is a person's verb, listed in the records group beside `status`. It
reads history as this checkout last fetched it, asks the forge at most once for
each pull request it has not read before, and writes only its forge cache in
the local tier.

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
named it refuses at exit 2, naming what to give. Named one commit, it traces
that commit instead.

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

The text prints one line per row: the commit, the pull request, the class, the
rule or a person's push's committer, the lane kind, the approval and where it
was read, and the issues and intents the row resolves and delivers, with the
commit that reverts it where one does. Then come the notes (how the range was
found, why the forge was not read for some row, why an answer was not kept, why
a first parent's mode could not be read) and the tip read. The machine form
prints the reader's listing with the same rows and stable keys. A filter keeps
the unattended rows alone, and another lists the reverted rows grouped by rule,
each group with a count, and the untraceable reverts in a group of their own.

## The trace

Named one commit, the verb prints the row that brought it onto the default
branch however far back, with its class and rule, and for a reverting commit
the rows it reverts. A commit the tip does not reach is said to be not on the
default branch, at exit 0; a name that names no one commit is refused at exit
2, and so is a trace given a listing's filter or start.

## What it prints of typed text

Every value a row carries is text someone typed: a subject, a trailer, a
committer's name, a forge login, a forge failure's text. The reader masks each
for a terminal and bounds it where it reads it, and the render masks and bounds
every value it prints again, each field at 200 bytes and each note at 600, so a
line is bounded whatever the reader hands it. The number of rows is bounded by
the history the reader takes, whose log it refuses whole past 64 MiB.

## What it touches

It fetches nothing and pushes nothing. Its one network act is the reader's: at
most one read through the forge client for each pull request the range names
whose answer its cache does not keep, asked in batches, and one read of a
squashed revert's pull request body for the forge's revert line. Each answer it
read is kept in the local tier, and no body is kept.

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
