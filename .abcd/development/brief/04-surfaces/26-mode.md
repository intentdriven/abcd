# `/abcd:mode` — Whose Answer the Loop Waits On, and the Away Mode

`/abcd:mode` holds two facts the agent loop is judged by. The first is the
waiting-on state behind the status line's badge: `managed`, `facilitator` or
`product-thinker`, stored per checkout at `.abcd/.work.local/mode`, printed by
bare `abcd mode` and set by `abcd mode <state>` (itd-200, spc-70). The badge,
who sets it and when it resets are described with the board that shows it, in
[`08-abcd.md`](08-abcd.md) ("The presence line" and "The badge is true at
every stop"). The second is the away mode, below: the person's word that they
are away and sessions should work on (itd-2610180912189125,
spc-2610180914469489).

The waiting-on state is not who merges. That is the repository's
`facilitator.mode` (`human` or `auto`, in `.abcd/config.json`), which the merge
gate reads (itd-2610150602547271, spc-2610161609350616). While it is `auto`,
the status-line row carries one more element after the branch,
`autonomous merging`; under `human`, which every fresh install starts with,
the row has no such element. The row reads the mode as the merge gate does:
the default branch's committed `.abcd/config.json` and the machine layer,
never the working tree's copy, so a branch that commits `auto` does not show
itself as autonomous. The read asks git only, with its questions inside the
row's own git time and killed at its end, and asks the network nothing. A mode
that cannot be read drops the element and says why on stderr, masked, as the
row's other elements do; a repository with no default branch yet shows
nothing and says nothing, having nothing to merge into. The element is
switchable in `~/.abcd.noindex/statusline.json` as `merging`, like the row's
other elements. It reaches the host only through the status line the person
consented to at install; nothing else in the host's settings calls abcd.

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
| `autonomous` | — | shipped |
| `back` | — | shipped |
| `queue` | — | shipped |

## The away mode

Beside the badge, and never a fourth word, the away mode records the person's
word that they are away and sessions should work on. Setting it records, once
for every working copy of the repository on the machine, who set it (the
holder name reservations resolve without a network read, else git's
`user.name`), from which working copy, when, and when it ends. The record
lives in `~/.abcd.noindex/away/<root-sha>/`, keyed on the root commit as the worktree
store is.

**When it ends.** The end is a clock time (its next occurrence, local time)
or a timestamp, or nothing, which leaves the record open-ended. Every record
ends seven days after it was set at most, so an end past that, or one already
past, is refused with nothing written. The person may also say that pushes
wait for their return; that word is kept in the record, narrows what sessions
do, and grants nothing. A prompt never ends the mode; its time does, or the
person's word that they are back.

**The queue and the return.** While the record holds, a session keeps a step
for the person's return instead of asking them: an act only the person may
approve. On their return the record is ended by word, which prints one plain
line per queued step, per item handed back and per recorded stop (a stop at
the host's cap labelled "possibly forced by the cap"), records who ended it and
from which working copy, and closes the record, on a record whose time has
passed as on one that holds.

**It grants no rights.** The person's own standing instruction stays what
decides whether a session commits, pushes, or opens or merges a pull request.

**The lock and the checks.** Every change to the record holds a lock file
beside it and re-reads the record inside the lock, so two writers never lose
an update and a check that read the record before the person came back never
writes it back; only ending the mode goes around a lock it cannot have
(below). The record and the lock are used only when each is a regular file
this account owns that no one else can write, within 64 KiB: one that fails
those checks, or does not parse, reads as not away, and queueing a step
refuses it, naming the file. Setting the mode refuses the same, except that it
carries a record merely past 64 KiB, letting its oldest lines go.

**Ending never refuses on the record.** The person's word that they are back
ends the mode whatever state the record or its lock is in. A record past
64 KiB is printed whole and kept with its oldest lines let go; one that does
not parse, or is not safely this account's, is replaced by a closed record,
printing nothing from it; and a lock not had within two seconds, or not safe
to take, is gone around, closing without it. Each is named on its own line.

**The badge while away.** While the record holds, parking the badge on the
product thinker or the technical facilitator is refused, naming
`abcd decide`, since a stop parked on a person who is away is a question
nobody answers.

Every refusal of the away verbs exits 2 with nothing written: an unreadable
or past end, one past seven days, a checkout without the local tier (checked
only when setting the mode), no checkout, a queued step with no record holding
or with no words, and, when setting the mode or queueing a step, a lock not
had within two seconds and a record or lock that fails the checks. Ending the
mode refuses only where there is no checkout, no home or no root commit to key
the record on. None makes a network request.

## A session off a profile

The away mode says sessions work on unattended, and every session that
coordinates unattended work is meant to start on a profile
(dcn-2611031722512037). A session the person started is warned and never refused
(dcn-2611041550129278), since no host shows a session's tools to any hook.

Setting the away mode warns; so does a build or a drain move while the mode
holds ([`34-build.md`](34-build.md), [`35-drain.md`](35-drain.md)), a first
session's join in any mode ([`27-implement.md`](27-implement.md)), and any of
those moves when the away state cannot be read. A managed build or drain move
whose away state can be read prints nothing.

The warning is one line on stderr whatever the output form, printed after the
move, opening `WARNING:`, worded as the setting's reading, naming what abcd cannot
check and asking to be passed on to the person. What remains owed about the line is
in dcn-2611050931229356. The exit code and the verb's behaviour do not change, and
a refused verb started nothing and owes no line.

**What decides.** One setting in the session's environment, resolved through its
links (`runner.SessionOnProfile`, in the core, which writes nothing; the front door
prints the line). A host that reports no such setting reads as off a profile. Every
doubt falls towards the line, since a line printed once too often costs a line: an
away record or store that cannot be read warns as if the mode held, and says why. A
repository with no root commit can hold no away record, and reads as not away.

**What it does not say.** This reads one setting and proves no wall, and the
setting's value changes nothing the session holds. A session on a profile could
hold other tools, and abcd has no way to see them. A host's report of a session's
tools would show the tools and still not bound its shell; what the account cannot
reach is a later step (itd-2610161621364075). A profile is a separate login and
settings on one operating-system account, and a stage's shell reads what the account
can. No abcd command starts a session on a profile yet.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd mode`

Sub-verbs: `abcd mode autonomous`, `abcd mode back`, `abcd mode queue`.

Flags: none.

### `abcd mode autonomous`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--hold-pushes` | bool |
| `--until` | string |

### `abcd mode back`

Sub-verbs: none.

Flags: none.

### `abcd mode queue`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
