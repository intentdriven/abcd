# `/abcd:sweep` — See What a Repository Left Lying Around, and Clear What Is Spent

`/abcd:sweep` is one verb over everything a repository leaves behind
(itd-2610222214476887, spc-2610230730269295): its worktrees, its local
branches, its merged branches on origin, its reservation branches and the
reports waiting in abcd's inbox. It is a person's verb, listed in the set-up
group beside `ahoy`, whose worktree reclaim it calls. Bare, it changes nothing
but its fetch: it lists every item with what the applying form would do, or
why it would leave the item. The applying form acts only on what is provably
spent, forces nothing and stops no process.

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

## What it reads first

The sweep fetches origin before it judges anything: origin's branches, pruned,
and the reservations. A fetch that fails refuses the run, because every
judgement after it would read origin as it stood at some earlier fetch. A
checkout with no origin fetches nothing, and then nothing can show a worktree
unreserved, so every worktree is kept.

## The rows

One row per item, in this order:

- **Worktrees**, in the worktree store and outside it, with the store's strays
  and the embargo store. A merged, clean worktree is removed through the
  store's own reclaim, which archives its notes first; there is no second
  reclaim. Its row names the ignored files outside its local tier that go
  with it unarchived, how many and the first few, or says they could not be
  listed; the person decides, and the applying form removes it all the same
  (dcn-2611161805215819). A dirty worktree no live session holds has its uncommitted work
  saved as a patch in the notes archive, and is kept. The embargo store is
  never read, saved or removed, and a stray is listed and never touched.
- **Local branches** with no worktree. A merged one is deleted the way git
  checks the merge again as it deletes.
- **Branches on origin.** A merged one is deleted under a lease on the tip
  that was judged, once nothing still needs it: no open pull request from it
  or into it, no reservation recording it, and no live session in a worktree
  on it. A reservation records no branch, only the copy that made it, so it
  holds the branch of a local worktree that copy is on; a reservation made on
  another machine or under another account holds no branch on origin, so
  for such a branch the reservation adds no hold: it is deleted once it is
  merged at the judged tip, unless an open pull request or another hold
  named here keeps it.
  A record id in a branch's name holds nothing by
  itself. The default branch, release branches, tags and
  abcd's own coordination branches are never deleted.
- **Reservations.** A spent one is cleared by the reservation sweep's
  workflow, which the applying form starts; the sweep never deletes one
  itself. Where it cannot start the workflow, it says when the daily run will.
- **Inbox reports.** A report is about abcd, so it is filed only in abcd's
  own checkout: there each waiting report is filed into the ledger with the
  automatic filer's remedy, so the drain leaves it until a person writes one.
  In any other repository every report is kept and the row says why. A
  report's text stays data.

Everything else is kept and named with the reason: a dirty or unmerged tree, a
lock another tool holds, a live session, a write within the last day, or an
item the sweep cannot classify. A live session is proven by a reservation, a
running process working inside the worktree or a build run naming it as a
lane. A check that could not be made holds every item it could have cleared,
and the result says so once, at the top.

## What it answers

The rows render as text, or as the core's own result for an agent, each row
with its kind, name, path, the state read, the action, the reason, and each
piece of evidence marked proven or a hint; applied, each row also carries what
was done. It exits 0 when the run acted on something (bare: when the applying
form would), 1 when it completed and acted on nothing, and 2 on a refusal or a
fault. Bare, it writes only under the machine's abcd home and the local tier,
the fetch aside; applied, it also removes worktrees and local branches, deletes
branches on origin, starts the reservation sweep's workflow and, in abcd's
own checkout, files the waiting inbox reports as ledger records.

## When abcd sweeps on its own

In a repository abcd manages, abcd runs the provably spent part of the
applying form itself (iss-2611150649557869, dcn-2611161331159806), with the
same proofs against a fresh fetch: once a landing sees its pull request
merged, whether a build's lane, a drain's, or a merged lane a build's start or
resume closes, at the end of a drain move that ends the drain, and at the end
of a drive a person did not stop and nothing killed or halted. A build run
ends at its last landing, which has swept already. It removes merged, clean
worktrees of the worktree store, saves a dirty one's work and keeps it, and
deletes the merged branches abcd made, locally and on origin: a branch named
exactly as the build loop names a lane, its prefix followed by a run id and a
lane id the loop mints (dcn-2611161459048806, dcn-2611161654034047). Any
other merged branch, a long-lived one at or behind the default branch, a
peer's freshly pushed one, a person's own branch under the lane prefix, or the
branch of a store worktree it removes, is listed with its reason and left for
the applying form, since no person saw a dry run first. A store worktree's
branch is no proof abcd made it: adding a worktree to the store can check a
person's existing branch out there. For the same reason a spent store worktree
holding any ignored file outside its local tier, which a removal deletes with
only the tier archived, is kept and listed for the applying form, naming how
many ignored paths it holds and the first few, and so is one whose ignored
files cannot be listed (dcn-2611161728290362). A file counts as ignored as the
store's own removal will delete it: by the repository's own ignore rules or
by the global ignore file git reads by default. A file only an ignore file
named in the person's own global git configuration matches reads as
untracked, since abcd's git reads no global configuration, so the worktree
is kept as changed and its work, that file included, saved as a patch in the
notes archive. At the last check before each removal it reads the ignored
files once more, and keeps a worktree in which one has appeared. Its line
names a kept worktree's ignored count and first path ahead of the worktree's
path, so the bound never cuts the warning off. It starts no reservation workflow and files no
inbox report: those stay with the applying form a person runs. The worktree
the running process is in, and any a live session holds, are kept by the same
liveness proof.

It is bounded at two minutes as a whole, well inside a lane's ten-minute
command bound: its fetch, its fetch of the reservations and each delete on
origin end with that bound or with a person's stop, and it starts no removal
past it. After a landing step, at a drain's end and at a drive's end it stops
between removals and never inside one; as a build starts or resumes it stops
only at the bound, and an interrupt there ends the process, perhaps during a
removal. Its local reads,
the merged judgement, the liveness probe's read of a worktree's changes and
the read of the ignored files a worktree holds, are short and run with no deadline of their own. What it did not reach is judged again,
with the same proofs, by the next landing's sweep or the drain's end. The act
it follows reports it on one line, bounded: what it removed and what it left,
counted, the first few rows in the sweep's row wording, then where the full
list is. A refusal before anything is judged says so; a refusal or fault of
that sweep is reported there and never fails the act it follows.

One setting turns it off: `sweep.automatic`, a boolean read through the
layered configuration reader from the repository's `.abcd/config.json` and
the machine's `~/.abcd.noindex/config.json`. It only restricts: false in
either layer holds, no layer turns the sweep on outside a repository abcd
manages, and a value that cannot be read reads as off, which the act's line
names.

## Where it is named

The shell guard refuses a delete of a branch or a worktree it cannot prove
safe, and that refusal names this verb as what settles a spent branch or
worktree (iss-2610241239492159), so an agent that meets the refusal is pointed
at the one verb that proves what it removes.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd sweep`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--apply` | bool |

<!-- surface-appendix:end -->
