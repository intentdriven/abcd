# `/abcd:reserve` — Reserve a Record on the Shared Remote Before Starting It

`/abcd:reserve` is the verb an agent calls before it starts on an issue or an
intent: it reserves the record on the shared remote as a branch nobody merges,
so a second person, a second computer or a hosted cloud session sees the
reservation before starting the same item (itd-2609091034175565,
spc-2610031753359849, adr-2610031352212697). It is an agent's verb (the product
thinker's ruling of 2026-10-12, "I suggest /abcd:reserve becomes an agent-only
verb"): it is listed in the agents-and-hosts block, its page is hidden from a
person's `/abcd:` menu. `abcd peers`, which lists the reservations, is an
agent's verb too; a person sees them on the board (bare `abcd`) and on a
record's page (`abcd <record-id>`), as [`08-abcd.md`](08-abcd.md) states.

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

## The forms

The verb takes one record, an issue or an intent in canonical id form, the two
families a build or a drain takes, or a release, `rel-vX.Y.Z` (the release
hold, below), and does one of two things: it reserves the record, or it
releases this copy's reservation of it. A third form, the holder query, takes
no record and is described under "The holder" below. A reservation names every
repository-relative path the work will change, one path to a flag, never split
on commas, and the footprint is required. Its lease is from one minute to 24
hours, two hours by default. A release takes no footprint, take-over or lease.
Bare `abcd reserve` refuses as a usage error and points at `abcd peers`, which
lists the reservations. The flags are in the appendix below.

## The holder

The holder is the person's GitHub username, worked out by the one holder
resolver in `internal/core/claim` (invariant 10). When git's identity, read
under the person's own configuration, is a person's (`user.name` and
`user.email` both set, and not a machine identity as the attribution gate
judges an author), the holder comes from the first of: `user.email` when it is
a GitHub no-reply address; `reservation.github_username` in the machine layer
of the configuration (`~/.abcd.noindex/config.json`), never the repository's
own; and the account of the forge token `gh` resolves from this session's
environment, which takes one
call to GitHub's user endpoint (`gh api user`, through the one door to
GitHub). When git's identity is not a person's (unset, or a machine's, as in a
hosted container whose clone comes with a vendor identity while its token is
the person's), the token alone names the holder, the machine key is skipped so
that the author's address and the login come from the same account, and that
account authors and commits the reservation as `<login>
<id+login@users.noreply.github.com>`. The identity git had is then shown
nowhere: in no output, commit, trailer, ref name or refusal. When no source
names the holder the act is refused, the refusal naming the class of the
failure and never the git identity it rejected.

Only an explicit act asks the user endpoint: a reservation, a renewal, a
take-over, a release, and the holder query. The renders that read reservations (the
board, a record's page and `abcd peers`) never do, and when only the token
could name the reader they treat the reader as unknown.

The holder query prints `would reserve as <login> (from <source>)`, the
source being `git`, `config` or `token`, or as JSON the document
`{"login", "id", "noreply", "source"}` (`id` and `noreply` when known). It may
make that one user-endpoint call in the same two cases as a reservation: when
git's identity is a person's but neither its address nor the machine layer
names the holder, and when git's identity is not a person's, whatever the
machine layer says. It takes no record id and no other flag, and reserves,
writes and pushes nothing. It exits 0 when the account is known and 2 when not.

## What it writes

The reservation is `refs/heads/abcd-reserved/<record-id>` on `origin`, a chain
of empty-tree commits written with `git commit-tree` and pushed with one plain
fast-forward push, the repository's hooks running. Each commit's trailers name
the record (`Abcd-Reservation`), the holder (`Abcd-Holder`, a GitHub username),
the working copy (`Abcd-Copy`, a random key kept in the checkout's local tier),
the footprint (`Abcd-Footprint`, one per path), the lease (`Abcd-Claimed-At`,
`Abcd-Expires-At`) and abcd's composed `Assisted-by: abcd:<version>` label; a
release adds `Abcd-Released-At`. The message is at most 4096 bytes, its
tombstone included, and passes the outbound policy. Nothing touches HEAD, the
index or the working tree, and a failed push leaves no reservation and nothing
in the checkout. The committed pre-push hook lets a push under the prefix
through without a preflight receipt only when every commit it moves has the
empty tree.

Every act fetches the reservation branches with pruning, and origin's default
branch, before it judges the tip. The acts are a reservation over no tip or a
released one, a renewal by the same person and copy, a take-over of the same
person's other copy, a take-over of another person's or an
unreadable reservation lapsed past its effective expiry or hold by at least
five minutes, and a release by tombstone. Moving the same person's
reservation from another copy needs the take-over flag; it never moves another
person's. A record resolved, won't-fixed,
shipped or superseded on origin's default branch counts as released whatever
its reservation says; when that branch cannot be read, the act goes on and the
output says the check was skipped.

The act is then judged against the rest of the fetched view. A reservation
that would take its holder past their cap of live reservations, counted by
username across every record, is refused naming the cap, how to raise it and
the reservations; a renewal, and a take-over of
the same person's other copy, of a reservation still live add none and are
never refused by the cap, while reviving a lapsed one adds one. A
footprint path that another live reservation's footprint shares, equal or one
a directory holding the other, is refused, unless that reservation is the same
person's from this copy, naming the deeper of the two paths,
the other record, its holder and its expiry; a renewal the verb makes names
its footprint and is checked the same way, so a footprint is never widened onto
another reservation, while the implement loop's renewal of a live reservation,
which keeps the tip's footprint, is not checked, and one that revives a lapsed
reservation is checked like a new one. The cap and the overlap are judged on the view just
fetched and are not compared-and-swapped, since each reservation is its own
branch: two people reserving at once can both pass them, so both are
best-effort under a race. A collision on one record is atomic, the plain push
being the compare-and-swap.

The cap is ten unless the person's own machine sets `reservation.max_live` in
`~/.abcd.noindex/config.json`, a whole number from 1 to 100
(`internal/core/claim/maxlive.go`, iss-2610170717515674). It stays one ceiling
per person, counted across all of their sessions and copies, so a person who
runs many sessions on one account raises it rather than releasing and
re-taking holds to make room. It is read from the machine layer alone: a
repository's `.abcd/config.json` that sets it is not read, so a checkout cannot
raise anyone's ceiling. A value outside the range, or a key under
`reservation` that nothing reads, refuses a new reservation naming the file,
never falling back to ten; a `config.json` that cannot be read at all
(writable by others, not valid JSON) is read past, ten applies, and the
refusal at the cap says why. The setting is read only by an act that adds a
reservation, so a bad value never stops a renewal or a release.

Reservations coordinate people who share push access in good faith; they are
not an access control. Any writer who can push can append to another holder's
branch, and the holder is a self-declared trailer, so the forge's push
permissions are the only boundary.

Expiry is the holder's clock judged by the reader's, and the five-minute
take-over margin is the only skew it absorbs: a holder whose clock runs behind
by more than half its lease plus the margin can be taken over between its own
renewals, and a reservation never renewed is open to a take-over once the
holder's clock is behind by more than its lease. A reservation dated more than
five minutes ahead of the reader counts from when that copy first fetched it,
so it holds at most a day from then. No bound read on the reader's side holds
for a slow clock: the commit date is written by the same clock, and the time
this copy first saw a tip cannot tell a holder's slow clock from a late fetch.

## The release hold

A session cutting a release reserves the release itself, `rel-vX.Y.Z`, on the
branch `abcd-reserved/rel-vX.Y.Z` (itd-2609151838312703 C7 and C8,
spc-2610150728396006 step 4, resolving iss-2609091037191879). While it is live,
every other session's land step refuses to arm its merge, naming the holder and
the release, and leaves the pull request open; `abcd peers` shows the line
"release v0.16.0 being cut by @maya: merges are held", judged by the merge
hold's own rule (`claim.JudgedHolds`, iss-2610160415393242): the cutting
copy's own release is shown as "you are cutting release v0.16.0 from this
copy: every other session's merges are held", and a release dated ahead whose
first sighting the checkout cannot keep is shown as not counted, in land's
notice's words, with `own` and `unsighted` on each `release_holds` row of
its machine-readable form. The subject is `rel-v` and three plain numbers,
with no leading zero and no suffix, so one release never makes two branches. It differs from a record's reservation in four ways:

- It takes no footprint, and refuses one: it holds every merge, not some
  files.
- It sits outside the per-person cap: it is neither counted toward a person's
  cap nor refused by it, and the live counts a listing shows leave it out.
- It holds for 24 hours unless a shorter lease is named, where a record holds
  for two. A cut waits on its checks, on a reviewer and on an approval, and
  the release tag is pushed by the release workflow after the merge, so a
  two-hour hold could lapse in the middle of a cut. A slower cut renews it by
  reserving it again from the same copy, which pushes a fresh day; without a
  renewal nothing holds past 24 hours from its claim, or from this copy's
  first fetch of one claimed ahead, the one bound every reservation has.
- It ends by itself when the tag `vX.Y.Z` is on origin as last fetched, as a
  record's ends when the record is finished on the default branch, or earlier
  when its holder releases it. A reservation of a release already tagged on
  origin is refused. The tags are read by the same fetch as the reservations,
  into `refs/abcd/release-tags/`, so the fetch never makes, moves or deletes
  a tag of the person's own and never lists a tag as a remote branch. Those
  are ordinary refs of the checkout all the same: a git log of every ref lists
  the commits they name, and a mirror push would push them.

The hold binds sessions, not people. The copy that holds it is not held, since
under the concurrent-sessions rule a session cutting a release has the final
say on what merges before its tag, and its own landing is that ruling; the
release pull request itself is merged by a person and meets no land fence.
Every other copy is held, the same person's other copy included, since the
hazard the hold guards (a merge between the release merge and its tag, which
makes the release workflow read the wrong commit) is the same whoever's
session merges, and its pull request is handed to the cutting session to merge
or hold. An unreadable release branch holds every merge for the default
two-hour hold from its commit date, as an unreadable reservation holds its
record, named as unreadable and never by its writer's text.

The release cut's ship sub-verb does not take or release the hold: the cutting session runs
this verb before the cut and the hold ends at the tag without it (the reasons
are in [`04-launch.md`](04-launch.md)). The land fence and `abcd land` judge
the hold through `claim.MergeHolds`, after their fetch: it fetches and pushes
nothing, and keeps in the local tier this copy's first sighting of each
release reservation dated more than five minutes ahead (or undated), whichever
fetch brought it, so such a release holds a day (an unreadable one two hours)
from that sighting at every path that judges a merge. Where no sighting can be
kept, in a checkout with no local tier or one whose tier cannot be written,
such a release would be counted from every reading and hold for ever, so it
is not counted as a hold: the judgement returns it as dated ahead, and the
caller names it in one notice line (the arming's note at the fence, a line on
stderr at `abcd land`). The tier is never made for it. An honest cut is
reserved on a clock within five minutes of the reader's, so this lets no
merge past one, with one limit: a cutting copy whose clock runs more than five
minutes fast is not counted by a checkout that cannot keep a sighting. `abcd
peers` renders the hold from its own view through `Snapshot.Holds`, with the
same line, as `abcd status` and the prompt hook do (`39-status.md`); the
holds read the release reservations' tips alone.

The hold stops a merge from being armed and nothing more. A merge armed, or
queued, before the release was reserved (a pull request with auto-merge on,
or an entry already in the merge queue) is not stopped by it and lands when
its checks pass. The cutting session checks for armed pull requests and
merge-queue entries after reserving and again before merging the release pull
request, and tells people working in the repository by hand to hold.

## Exits

Exit 0 when the act reached origin, or a release was already made. Exit 3, the
back-off, when another person's reservation is live or lapsed under the
margin, an unreadable one is within its hold, another push won the race, or the
same person's other copy holds it and no take-over was asked for, or a
footprint path is in another live reservation's footprint. Exit 2 on the cap
and on every other refusal and failure. The JSON document carries `act`, `record`, `holder`,
`copy` (`this` or `another`, never a copy's key), `footprint`, `claimed_at`,
`effective_expiry`, `commit`, `replaced` for a take-over and
`derived_unchecked` when the default branch could not be read.

## Where this sits

- The intent and its decisions: itd-2609091034175565; the design record:
  spc-2610031753359849, whose spec-2 steps add the build, drain and render
  integration.
- The listing: `abcd peers` ([`08-abcd.md`](08-abcd.md)).
- The release hold: itd-2609151838312703 (C7, C8), spc-2610150728396006
  step 4; the land fence that reads it is in
  [`27-implement.md`](27-implement.md).
- The plugin surface: `commands/reserve.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd reserve`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--footprint` | stringArray |
| `--holder` | bool |
| `--lease` | duration |
| `--release` | bool |
| `--take-over` | bool |

<!-- surface-appendix:end -->
