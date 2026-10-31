# `/abcd:announce` — Post a Fixed-Form Announcement on the Shared Remote

`/abcd:announce` is the verb a session calls to tell every other session what
it is doing: on the same computer, under a colleague's account on another
computer, or in a hosted cloud session (itd-2609151838312703,
spc-2610150728396006, adr-2610150729333337). It is an agent's verb, beside
`/abcd:reserve` (the product thinker's ruling of 2026-10-15, decision 21 of the
intent): it is listed in the agents-and-hosts block, and its page is hidden
from a person's `/abcd:` menu. An announcement informs and grants nothing; what
is really a hold is a reservation.

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

## The form

The verb takes a kind and an item and nothing else. The kind is one of
`merging`, `merged`, `asking`, `answered` and `handed-over`. The item is an
issue or an intent in canonical id form, or a pull request, `#N` or the bare
number, which is stored as `#N`. An announcement stands for 30 minutes by
default, from one minute to 24 hours, held to the whole second below the
expiry asked for. It carries no free text: a further word
after the item is refused, and a question points at the item, where the
conversation happens. Bare `abcd announce` refuses as a usage error naming the
kinds. The flag is in the appendix below.

## What it writes

The announcement is one empty-tree commit on the sender's own status branch,
`refs/heads/abcd-status/<holder id>` on `origin` (the holder id a reservation
names, `h_` and 32 lower-case hex characters, kept in
`~/.abcd.noindex/holder-id`; iss-2610290809526033), one branch per person, written with `git commit-tree` and pushed with one plain
fast-forward push, the repository's hooks running. Its subject is
`abcd announcement: <kind> <item>`, and its trailers carry the kind
(`Abcd-Announcement`), the item (`Abcd-Item`), the sender (`Abcd-Sender`, the
holder id, never the person's login), the working copy (`Abcd-Copy`, the key a
reservation uses), the time sent (`Abcd-Sent-At`), the expiry
(`Abcd-Expires-At`) and abcd's composed `Assisted-by: abcd:<version>` label,
and nothing else. The commit is authored as the person: their own git identity
when it is a person's, and otherwise the account the session's forge token
belongs to. Nothing touches HEAD, the index or the working tree. A status
branch made before the holder id is named by its sender's GitHub username in
lower case and carries it as `Abcd-Sender`; such a branch is still read, and
none is pushed any more. A copy on an older release reads only the
login-named branches, so it does not see an announcement on a holder-id branch
until it updates; a person running two versions sends from the new one.
Pushing no login has one cost, and it is a stated limitation: the forge's
record of who pushed names an account, and a holder id names none it could be
matched against, so the sender of an announcement on a holder-id branch is
shown as claimed, never as verified (`abcd status`, C17). Only a branch named
by a username, made before the holder id, can still show its sender as
verified.

When the item is a record and this clone names no default branch of origin,
the verb first asks origin which branch is its default (`git ls-remote
--symref origin HEAD`, which writes nothing). It then lists the status
branches on origin (`git ls-remote`), then fetches the first 500 whose names
are holder ids or lower-case GitHub usernames, in origin's order and with the
sender's own among them, those past the 500 counted and not fetched, each by
its own refspec, with origin's default branch in one fetch, and the status branches
alone in a second when the default branch cannot be read. It never fetches the
namespace by a glob: a branch whose name differs from another only in case
(`abcd-status/Octo` beside `abcd-status/octo`) would break such a fetch on a
case-insensitive filesystem, so a name that is neither a holder id nor a
lower-case username is not fetched at all; and it never names a branch the listing did not show,
since a fetch of a branch origin does not hold fails. Each refused push is
followed by one more listing and fetch. A push git reported refused that the branch
now holds landed, and is reported sent. A push that lost a race to another of
the same person's sessions is rebuilt on the tip that won and pushed again, up
to three times, and then refused as `not sent:` with the reason; a push refused
while the tip stands is refused at once. Nothing is
ever forced. The committed pre-push hook lets a push under `abcd-status/`
through without a preflight receipt only as a fast-forward of commits that
change no file, refuses a deletion, and gates every other push as any branch
is; a hosted session runs no hook, so there the ruleset on
`refs/heads/abcd-status/**` and abcd's reader are the guards
(adr-2610150729333337).

## What it refuses

Every refusal pushes nothing, except `not sent:`, the one refusal made after
pushes, each of which origin refused. A push made whose outcome cannot be read
back (the fetch after it failed, or the rebuilt commit could not be written) is
not a refusal: exit 2 says a refused push may still have reached origin.

- A kind outside the list, an item outside its grammar and an expiry outside
  its bounds are refused before git runs, naming the field and the limit.
  Every announcement is composed, then read back through the same validator a
  reader holds it to: its lead is exactly its subject, and every line of its
  final paragraph is one of its trailers, written exactly, its `Assisted-by`
  abcd's own `abcd:<version>` label and no other value.
- A record not on origin's default branch as just fetched is refused as not
  public yet, read by the same tree reader the reservation's derived release
  reads by, and so is any record when no default branch was fetched or the
  fetch could not read it.
- A record the embargo store on this machine holds for an advisory not yet
  published is refused through the embargo check, which also reads the whole
  message; a store that exists but cannot be checked refuses every
  announcement.
- No GitHub account to send as is refused before any fetch or push, naming
  the failure's class and never the identity it rejected, and so is a
  holder-id file the reservation's holder would refuse. The account is
  resolved by the one resolver a reservation's holder is
  (`internal/core/claim`), in the same order: user.email when it is a GitHub
  no-reply address, then `reservation.github_username` in the machine
  configuration, then the session's forge token, asked through one call to
  GitHub's user endpoint whose account it is. When git's identity is not a
  person's (unset, or a machine's as the attribution gate judges an author),
  the token alone is used, its account authors and commits the announcement,
  and the identity git had appears in no output, commit, trailer or ref. The
  working copy and the remote named `origin` are checked first, so a checkout
  refused on its own facts never asks GitHub.

## abcd's own pushes announce first

The pushes abcd makes for an item post an announcement before they push, so a
colleague's session sees `merging <item>` before the work is on `origin`
(iss-2610161645224876, dcn-2610161645360719, dcn-2610170655202547): `abcd land`
for each issue, intent or pull request its branch's own commits name in a
`Refs:`, `Resolves:` or `Delivers:` trailer (dcn-2610170704191129), and the
landing of a build or drain lane (the implement loop's land stage, which
`abcd build` and `abcd drain` drive) for the lane's own record. Each goes through
`claim.AnnounceBeforePush`, which calls `Announce` with the kind `merging`:
the same checks, the same fetch and the same push, and the push it was made
for waits until it has reached `origin`. One the person already sent, of
that kind and item and standing a minute more, is not sent again when it is
the sender's newest announcement of that item (read from the sender's own
status branch as just fetched, at most its newest 100 commits, and read again
after a lost race); under a newer one of the item, `merged` say, it no longer
stands, since a reader shows only the newest, and `merging` is sent anew. A
record not yet on the default branch is not announced and the
verb says so; any other failure refuses that push, nothing pushed
(dcn-2610170704189712). When a branch names several items and a later one
fails, the announcements already sent for the earlier ones stay standing, and
a re-run resumes without sending them again while they stand. A push a person types by hand with `git push` posts
no announcement, and no hook posts one: the pre-push hook is unchanged.

## Exits

Exit 0 when the announcement reached origin; exit 2 on every refusal, and when
it was not sent. The JSON document carries `kind`, `item`, `sender`, `branch`,
`sent_at`, `expires_at`, `commit` and `attempts`.

## Where this sits

- The intent and its decisions: itd-2609151838312703; the design record:
  spc-2610150728396006; the reader, `abcd status` and the prompt hook's
  block: [`39-status.md`](39-status.md).
- The trust rule: adr-2610150729333337; the ruleset mirror:
  `.abcd/work/rulesets/status-protection.json`.
- The plugin surface: `commands/announce.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd announce`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--expires-in` | duration |

<!-- surface-appendix:end -->
