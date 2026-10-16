# `/abcd:status` — Read Everyone's Status on the Shared Remote

`/abcd:status` is the people's view of what everyone else is doing on the
project (itd-2609151838312703, spc-2610150728396006 step 3, decision 21 of the
intent): the announcements every session posted with `/abcd:announce`, on this
computer, under a colleague's account on another computer or in a hosted cloud
session, then who holds which reservation and each release being cut. It is a
person's verb, listed in the records group. The same announcements reach a
running session through the prompt hook, once each, from disk alone (below).

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

## What it reads

The verb reads the remote once. It lists the status branches on `origin`
(`git ls-remote origin refs/heads/abcd-status/*`), then fetches, in one
command, the reservations with `origin`'s release tags, each status branch
whose name is a GitHub username in lower case, by its own refspec, and
`origin`'s default branch, and stamps when in the local tier. A branch under
`abcd-status/` by any other name is never fetched: two names that differ only
in case make every fetch fail on a case-insensitive filesystem, and the
no-deletion ruleset would keep the bad branch for ever (decision 10), so the
name is kept and shown as doubtful. Then it asks the forge, through `gh api`
(the repository's activity record, `GET /repos/{owner}/{repo}/activity`, one
call per status branch), who pushed each standing announcement it has not
asked about before (C17). It pushes nothing.

## The intake

One intake serves the verb and the prompt hook. It walks each status branch
from where this checkout last read it (the bounded history walk of step 1),
reads every commit the walk added that this checkout has not read before, in
one batch of two `git cat-file` commands whatever their number, each through
the announcement's own validator, and keeps in `.abcd/.work.local/status/`
the announcement's validated fields and when this checkout first fetched it.
The commit's raw text is never kept and never shown. An announcement stands
for the life it was sent with from that first fetch, not by the sender's
clock (C9); one whose own expiry passed more than an hour before the first
fetch is history a first read brought in, and is not counted anew.

A sender is shown as verified when the forge's record of who pushed names the
branch's owner, as doubtful when it names someone else, and as claimed when
the forge was not asked or cannot say: the native default, with the forge
lookup the adapter (invariant 2). These are doubtful, each with its reason,
while every other branch is still read (C11): an announcement whose sender is
not its branch's owner; a commit that does not read as an announcement, such
as one that changes a file, shown only by its branch and the reason; a branch
that moved back or was rewritten, passed a merge, or gained more than one read
covers since it was last read here, or holds more than one read covers at its
first read here ("continuity not verified"); and a branch
whose name is not a lower-case username.

## What it shows

The text form lists the announcements newest first, by when this checkout
first fetched each, then by the time it carries, each by the line the prompt
hook shows it by, `@maya (verified): merging #1012, for 30 min`, with the time
sent; then each doubtful branch; then the reservations block `/abcd:peers`
lists, release holds included, each by the line the land fence counts it by;
then the line that an announcement informs and grants nothing. A failed fetch
lists the status as last fetched, with its age and a line per namespace saying
it is unread since then and why ("status unread since 09:12 UTC: <reason>"),
and an empty list is said ("none standing") only after a fetch that
succeeded (C10). The JSON document carries `status_view`, `announcements`,
`doubtful`, `unread`, `senders_unchecked`, `grants`, and the reservations
block's own fields.

## The prompt hook's block

The prompt hook appends a `## STATUS` block to the text it injects
when this checkout holds something it has not shown: a failed fetch per
namespace, a release being cut ("release v0.16.0 being cut by @maya: merges
are held", C8), a doubtful branch, and the newest announcement per item, at
most ten lines, then a count of the rest and the command that lists them all
(C18). Each line is shown once, kept by commit or by what it says, and the
block is dated by the last stamped fetch ("as of the last fetch 4 minutes
ago", C12) and ends with the line that an announcement grants nothing (C6). It
reads only the local tier and touches no network, and the read runs under a
deadline inside the hook's one-second budget: when the deadline passes first,
the block says the status is unavailable rather than show nothing, and the
prompt is never held. A checkout with no remote named `origin`, or without its
local tier, shows nothing. The block is emitted on the rules loader's error
path too: the two share the hook, and a `rules.json` that will not load does
not switch the status off.

## Exits

Exit 0 when the fetch succeeded, and in a checkout with no remote named
`origin`; exit 1 when the fetch failed, the failure named on stderr; exit 2
outside a git checkout or with an operand.

## Where this sits

- The intent and its decisions: itd-2609151838312703; the design record:
  spc-2610150728396006.
- The writer: [`38-announce.md`](38-announce.md); the reservations block and
  the release hold: [`37-reserve.md`](37-reserve.md).
- The plugin surface: `commands/status.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd status`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
