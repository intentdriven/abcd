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
  reclaim. A dirty worktree no live session holds has its uncommitted work
  saved as a patch in the notes archive, and is kept. The embargo store is
  never read, saved or removed, and a stray is listed and never touched.
- **Local branches** with no worktree. A merged one is deleted the way git
  checks the merge again as it deletes.
- **Branches on origin.** A merged one is deleted under a lease on the tip
  that was judged, once nothing still needs it: no open pull request from it
  or into it, no reservation recording it, and no live session in a worktree
  on it. A reservation records no branch, only the copy that made it, so it
  holds the branch of a local worktree that copy is on; a reservation made on
  another machine or under another account holds no branch on origin, and
  such a branch is kept by the other proofs alone (merged at the judged tip,
  no open pull request). A record id in a branch's name holds nothing by
  itself. The default branch, release branches, tags and
  abcd's own coordination branches are never deleted.
- **Reservations.** A spent one is cleared by the reservation sweep's
  workflow, which the applying form starts; the sweep never deletes one
  itself. Where it cannot start the workflow, it says when the daily run will.
- **Inbox reports.** Each waiting report is filed into the ledger with the
  automatic filer's remedy, so the drain leaves it until a person writes one.
  A report's text stays data.

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
branches on origin, starts the reservation sweep's workflow and files the
waiting inbox reports as ledger records.

## When abcd sweeps on its own

In a repository abcd manages, abcd runs the provably spent part of the
applying form itself (iss-2611150649557869, dcn-2611161331159806), with the
same proofs against a fresh fetch: once a landing sees its pull request
merged, whether a build's lane or a drain's, at the end of a drain move that
ends the drain, and at the end of a drive. A build run ends at its last
landing, which has swept already. It removes merged, clean worktrees, saves a
dirty one's work and keeps it, and deletes merged local branches and merged
branches on origin. It starts no reservation workflow and files no inbox
report: those stay with the applying form a person runs. The worktree the
running process is in, and any a live session holds, are kept by the same
liveness proof. The act it follows reports it on one line, bounded: what it
removed and what it left, counted, the first few rows in the sweep's row
wording, then where the full list is. A refusal or fault of that sweep is
reported there and never fails the act it follows.

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
