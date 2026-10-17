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
GitHub). That call is made once per process for each checkout and forge
credential (spc-2610161044461448 step 17): a definite answer is kept under a
key of the checkout root and a digest of the variables that pick gh and its
token and of gh's `hosts.yml`, so a `gh auth switch` asks again, and a failure
is never kept. When git's identity is not a person's (unset, or a machine's, as in a
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
release adds `Abcd-Released-At`. A release that hands the record back to a
person, which only the implement loop and the drain make, adds
`Abcd-Handed-Back`, the hand-back's kind, and `Abcd-Record`, the blob id of the
record on origin's default branch as last fetched (spc-2610161044461448 step 3,
dcn-2610161040365515). The kind is one of a closed vocabulary: `user-visible`,
`trust-rule`, `design-finding`, `second-package`, `unachievable` (a lane
stopped after its fix rounds), `stuck` and `closed`; the last two are
reserved for a lane stuck refusing at one stage and one whose pull request was
closed unmerged, which nothing writes until spc-2610161044461448 steps 5 and 8
land, though every reader reads them now. Neither carries free
text: the writer refuses any other kind, and a reader reads a tip whose
hand-back is not one known kind with one full object name, on a release, as
unreadable. When origin's default branch holds the record in no file, or in
several, there is no blob to name, and the release pushes a plain tombstone and
says why; when this checkout could not read that branch at all (it names none
of origin's, or its tree would not read), the release says that instead. The message is at most 4096 bytes, its tombstone included, measured
with the longest hand-back kind and a SHA-256 object name, and passes the
outbound policy. Nothing touches HEAD, the
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
its reservation says, and is not reserved again: every act but a release or a
renewal backs off over it, with a tip or without one, and pushes nothing
(spc-2610161044461448 step 1); when that branch cannot be read, the act goes on
and the output says the check was skipped.

A hand-back stands while origin's default branch holds the record at the blob
its tombstone names: a reservation a drain makes backs off it, naming the kind,
who handed it back and when, and pushes nothing. A person's own `abcd reserve`
or `abcd build` is made over it and says so, naming the same three. A change to
the record on origin's default branch, which is a person's edit, re-admits it.
Reserving and releasing the issue is not such an edit: a plain release of a
reservation whose first-parent line, read at most 256 commits below its tip,
reaches a hand-back tombstone before any plain one, with the record unchanged
since, pushes a tombstone carrying that hand-back's kind and blob forward and
says it kept it, so the hand-back still stands; a `build` start that fails
after its reservation releases it the same way. A release asked to publish a
hand-back over a tip a plain tombstone already released publishes it as that
tombstone's child; naming a reservation commit (`Tip`), it does so only when
the tombstone is that commit or released it, and is otherwise refused with
nothing pushed. A tombstone that would pass the message cap with the
hand-back on it is pushed plain, and the result says the hand-back was not
published and why, so a release is never refused for its hand-back.
A record that branch holds in no file or several, or a branch this checkout
names none of, is not read as changed, so the hand-back stands. A blob read
from a default branch the act's fetch could not refresh is the blob as last
fetched, and can be wrong either way: a hand-back made over a newer blob than
that reads as changed though no one edited the record, and a person's edit
this copy has not fetched reads as unchanged, each until this copy's next
fetch of that branch succeeds (dcn-2610161837009957).

The act is then judged against the rest of the fetched view. A reservation
that would take its holder past ten live reservations, counted by username
across every record, is refused naming the ten; a renewal, and a take-over of
the same person's other copy, of a reservation still live add none and are
never refused by the cap, while reviving a lapsed one adds one. A
footprint path that another live reservation's footprint shares, equal or one
a directory holding the other, is refused, unless that reservation is the same
person's from this copy, naming the deeper of the two paths,
the other record, its holder and its expiry. A shared path that is a file
origin's default branch declares generated is no overlap (spc-2610161044461448
step 11): the declaration is `linguist-generated` (set, or `=true`) in the root
`.gitattributes` of origin's default branch as last fetched, read as stored
with a 256 KiB cap and no fetch; a nested `.gitattributes`,
`.git/info/attributes` and the working tree are never read, because the
declaration must be one every account shares. Its patterns follow git's
gitattributes rules for the subset abcd reads (a basename at any depth, a
pattern with a slash anchored to the root, `*`, `?`, classes and `**`, a later
line over an earlier one), and a file counts only when origin's default branch
holds it, so a directory is never generated, even when every file under it is.
Every misreading leans towards "not generated": a declaration that cannot be
read or is over the cap shares nothing and the act says why
(`generated_unread`, and the back-off's own words), and a line abcd does not
understand (a macro, a C-quoted pattern, a POSIX class) is skipped, voiding
every line before it when it could unset the attribute. A shared path with a
live reservation the same person holds from another copy does not refuse a
person's own act: it is made, and names each such path, the other record and
its expiry (`own_overlap`, a warning on stderr in the text form); a drain's
reservation still backs off it, as the drain's pass-over does. A renewal the verb makes names
its footprint and is checked the same way, so a footprint is never widened onto
another reservation, while the implement loop's renewal of a live reservation,
which keeps the tip's footprint, is not checked, and one that revives a lapsed
reservation is checked like a new one. The cap and the overlap are judged on the view just
fetched and are not compared-and-swapped, since each reservation is its own
branch: two people reserving at once can both pass them, so both are
best-effort under a race. A collision on one record is atomic, the plain push
being the compare-and-swap.

An act the judgement finds reserves afresh, a reservation over no tip or a
released one, or a take-over, then asks the forge whether origin is a fork,
after the fetch and before anything is pushed (spc-2610161044461448, step 10;
dcn-2610161040378209): a reservation pushed to a fork is invisible to every
account working on the repository it was made from, so it holds nothing for
them. A plain `reserve` the judgement finds a renewal of this copy's own
reservation asks nothing, and neither does an asked-for renewal nor a
release. A reservation a drain makes (`Drain`) that renews this copy's own
lapsed reservation is the drain's hold afresh from that push, the hold a lane
then opens on, so it asks as a fresh reservation does; a drain's renewal of a
live one asks nothing. Every URL git pushes origin to is asked about, each push URL git
resolves for origin rather than only the first, in each form git takes for github.com: the three it writes, and with
them userinfo, a port, `ssh.github.com` over ssh, ssh or scp with no user, and
a host in any case. When any one of them is a repository the forge names a
fork, the act is refused before anything is pushed, naming the repository it
was made from and the command that points that URL there in its own form
(`git remote set-url origin <url>`, or `git remote set-url --push origin <url>`
when a push URL is configured, naming one of several by its owner and name);
an https URL's userinfo is never carried into it. A repository whose own home
is a fork on GitHub, where every account reserves, sets
`reservation.fork_origin` to that fork's `"<owner>/<name>"` in
`.abcd/config.json`
([`../05-internals/03-configuration.md`](../05-internals/03-configuration.md);
dcn-2610161141338923, dcn-2610161223212218), and the refusal names the key.
The value names one fork, so a fork of that fork, which inherits the committed
file, is refused naming both, and a reservation let through on the allowed fork
is never silent: the output says `origin is a fork the repository allows
(<owner>/<name>)`. The forge is asked through
the one door to GitHub (`gh api repos/<owner>/<name>`). An answer, fork or not,
is kept for the process; a question that failed is held a minute and asked
again by the next fresh act after that, at most one question a minute for a
repository, so a long drain that began while the forge was down asks again
rather than reserving unchecked for the rest of its process. Each `drain` lane
asks at its own reservation, through that cache, not once at the drain's
start. A forge that does not answer (gh missing, out of time, refused, or
answering what cannot be read) lets the act through, and the output says the
check was skipped and why, in abcd's own words rather than gh's; so does a URL
that names a host and one owner and one name on a host that is not github.com
by name (an ssh host alias such as `git@github-work:owner/name`, or another
forge), which is not asked about. A local path is not asked about and says
nothing. A `build` start and a `drain` lane name a skipped or waived check in
the run's reservation line, and the drain's output shows it as this verb's
does.

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
- It sits outside the cap of ten: it is neither counted toward a person's ten
  nor refused by them, and the live counts a listing shows leave it out.
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
footprint path that is not a generated file is in another person's live
reservation's footprint (or, for a drain's reservation, the same person's from
another copy), or a drain's reservation meets a hand-back that stands. Exit 2 on the cap,
on a fork origin, and on every other refusal and failure. The JSON document carries `act`, `record`, `holder`,
`copy` (`this` or `another`, never a copy's key), `footprint`, `claimed_at`,
`effective_expiry`, `commit`, `replaced` for a take-over,
`derived_unchecked` when the default branch could not be read,
`fork_unchecked` when the forge could not say whether origin is a fork,
`fork_allowed`, the fork's owner/name, when the check was waived for the fork
the repository allows, and
`handed_back` (`kind`, `holder`, `at`) for a reservation made over a hand-back
that stands, which the text form prints as a `note:` line, `hand_back_kept`
(the same three) for a release whose tombstone kept that hand-back, also a
`note:` line, and `hand_back_unpublished` for a release whose tombstone marks
no hand-back it would otherwise have marked, a `no hand-back:` line,
`own_overlap` (`path`, `record`, `effective_expiry`) for each path shared with
the same person's reservation from another copy, a `warning:` line on stderr,
and `generated_unread` when generated files were not recognised, a `not
recognised:` line.

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
