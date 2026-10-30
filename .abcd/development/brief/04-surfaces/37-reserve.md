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
on commas, and the footprint is required. Each path should name a file. A
path naming a directory holds off every other reservation of any file inside
it, so a lane holding a whole package serialises every unrelated lane that
touches one file there (iss-2610230016544142). A footprint path naming a
directory git tracks is still reserved, never refused, and the act names it,
with how many tracked files lie beneath it and the branch they were counted on
(origin's default branch as last fetched, or the checkout's own head for a
directory only it holds), telling the person to name the files instead
(`directory_footprint`, a warning on stderr in the text form). A path neither
holds as a directory, such as a file the work will add, draws nothing. A path
named twice is one path, written once where it was first named
(iss-2610290507514841). A reserve that renews this copy's own reservation, or
takes over the person's own from another copy, writes its footprint in place
of the one held, and, when that reservation was live, names each held path it
leaves out, neither named again nor beneath a directory it names, which no
longer holds anything off (`dropped_footprint`, a warning on stderr in the text form,
iss-2610290507512835). Its lease is from one minute to 24 hours, two hours by
default. A release takes no footprint, take-over or lease.
Bare `abcd reserve` refuses as a usage error and points at `abcd peers`, which
lists the reservations. The flags are in the appendix below.

## The holder

What a reservation pushes names its holder by the person's holder id
(iss-2610290809526033): `h_` and 32 lower-case hex characters, 128 random bits
minted once, on first need, into `~/.abcd.noindex/holder-id` with an exclusive
create at mode 0600, so two racing first needs keep one id. The mint judges
each folder it makes or finds as the read below judges it, so a
`~/.abcd.noindex` another account can write refuses before anything is minted
or pushed. A first need that loses the race and still finds the file empty
after a short wait refuses with a retry, and never advises removing the file:
the winner may already have pushed under the id it is writing. The act that
mints says so once, on stderr, naming the id and the file: holds made under an
earlier id of the person's read as someone else's until they lapse, and the
cap counts them apart; the next act, which reads the id kept, says nothing. It is no hash of
anything, so it says nothing of the account; the underscore cannot occur in a
GitHub username, so it never collides with the login an older reservation
names; and lower case keeps a status branch named by it apart from every other
on a case-insensitive filesystem. It is read back only when the file is a
regular file the account owns and no other account can write, through the
guard every home declaration is read through; a file that fails that, or holds
no id, refuses the act and is never replaced. A HOME inside a git working tree
that is not the account's own is judged as the machine layer is: no holder id
is read from it or minted into it, so an act there refuses before anything is
pushed, and a render reads neither an id nor a login there (the login may come
from that home's machine configuration), so it names no hold as the reader's,
a login-named hold made before the id included. No trailer and no ref name
abcd pushes carries the person's login. The commit author and committer fields
carry the person's git identity when it is a person's, and otherwise the login
and no-reply address of the account the forge token resolves to (below), by
ruling; and holds already pushed under a login are not rewritten.

The id is read first from the environment variable `ABCD_HOLDER_ID`, then from
`~/.abcd.noindex/holder-id`, and only then minted. The variable is for a
session whose home is not kept, a hosted or ephemeral one: set in that
environment's own settings to the id the holder query prints on a machine
whose id the person keeps, it is the id, and it is never written to the file.
Set to anything but one id (`h_` and exactly 32 lower-case hex characters), it
refuses every act that needs the id, naming the variable and never its value,
and neither the file nor a mint stands in for it; a render reads no id from it.
A hosted session that does not set it mints a new id at each fresh home: the
holds its earlier sessions made read as another holder's until they lapse (no
take-over moves them), and the cap counts each session's holds separately.

A hold is the person's own when its holder is their kept id, an id listed in
`~/.abcd.noindex/holder-ids`, which is where a person declares the ids of
their other machines and accounts, or, on a hold made before the holder id,
the login the resolver below yields, compared without regard to case. The
declaration holds one id per line, `h_<32 hex>` alone or followed by the
GitHub login of the account that id is the person's on
(`h_<32 hex> <login>`, whitespace between, `#` starting a comment): an id with a login
beside it is paired with that login, one without is unpaired. It is read only
from the account's own home, never from a repository's tiers, through the
guard every home declaration is read through: a regular file the account owns
that no other account can write, through no symbolic link, within 64 KiB.
Absent, it declares none. A file failing any of that, a line that is not one
id and at most one login, and an id declared twice with two pairings each
refuse every act that reads it, naming the file and the line's number alone,
never its contents, with nothing pushed, while a render reads no declared id
from it and still reads the kept id and the login, and says so: `abcd peers`,
`abcd status` and the holder query each write one line on stderr
naming the file, the class of the refusal and that no declared id is read,
never the file's contents or a login, and carry the class in their JSON
documents as `holder_ids_refused` (`not_regular_file`, `writable_by_others`,
`owned_by_another_account`, `behind_symbolic_link`, `folder_exposed`,
`too_large`, `unreadable` or `malformed`). The paired login is read
and compared on this machine alone: it is never written into a trailer, a ref
name, a record or a comment, and no output names it. Every
"yours" judgement reads it this way: the renewal and release of one's own
reservation, the other-copy back-off and warning, the cap's count, the
ceiling, the land fence's own-release check and the renders that say "you".
A renewal or release of an old reservation of the person's writes the holder
id; no commit carries the login beside it, not even for one release. The
listing of each holder's hand-backs of the last day (`abcd peers`) goes by the
name each tombstone carries, not by person, since it knows no one else's ids:
one person's hand-back made before the holder id and one made after list as
two holders. That is accepted as the change's cost: no tombstone is named by a
login any more, so it lasts at most the day's window after the update. The
release hold's withdrawals are the person's own by the same judgement, and a
release branch is still held to the forge's record of who pushed each commit:
a commit is read as its pusher's when it names them by username, or names a
holder id the reader does not own, which no record of pushes can be matched
against. The reader's own ids are held to an account: their kept id counts
only when the reader's own login pushed the commit, a declared id paired with
a login counts only when that login pushed it, and an unpaired declared id
counts only when the reader's own login pushed it, as the same person's other
home under one login, minting an id of its own, does: no pairing is needed
for that. A withdrawal under any of the reader's ids, the kept one included,
is acted on only when the reservation that first recorded it, held under
that id, was pushed by the account the id is held to (the reader's own login
for the kept id and an unpaired one, the paired login for a paired one); a
later push that carries it says nothing of who first recorded it, so an
entry another login first recorded under one copy's kept id, carried by a
second copy of the same person that declares that id, is never read as the
first copy's own.

Across versions the change fails safe. A copy on an older release reads the
new holds as held until it updates: its reader takes a holder to be a GitHub
username, and no holder id is one, so it reads each new reservation as
unreadable, which it holds until the default lease from the commit's date and
the take-over margin have passed, backing off and never writing over it. A
person running two versions renews only from the new one: the older copy
refuses to renew a hold it cannot read, while the new one reads both the holds
it made and the login-named holds the older one made, and writes the holder id
when it renews them. The older reader is not run in the tests; what pins this
is that no minted id passes the username rule and that a reader meeting a
holder it cannot read backs off until the hold ends and leaves origin's tip as
it stands.

The person's GitHub username is still worked out, locally and never pushed, by
the one holder resolver in `internal/core/claim` (invariant 10). When git's identity, read
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
the person's), the token alone names the account, the machine key is skipped so
that the author's address and the login come from the same account, and that
account authors and commits the reservation as `<login>
<id+login@users.noreply.github.com>`. The identity git had is then shown
nowhere: in no output, commit, trailer, ref name or refusal. When no source
names the account the act is refused, the refusal naming the class of the
failure and never the git identity it rejected.

Only an explicit act asks the user endpoint: a reservation, a renewal, a
take-over, a release, and the holder query. The renders that read reservations (the
board, a record's page and `abcd peers`) never do, and when only the token
could name the reader they treat the reader as unknown.

The holder query prints `would reserve as <login> (from <source>)`, the
source being `git`, `config` or `token`, then `holder id: <id> (kept in
~/.abcd.noindex/holder-id)`, or `(from ABCD_HOLDER_ID)`, with the same
statement of what the id keeps out of what is pushed (no trailer or branch
name carries the login; the author and committer carry the git identity when
it is a person's, otherwise the token account's login and no-reply address;
holds already pushed are not rewritten), and, when any are
declared, `also yours: <ids>`; or as JSON the document `{"login", "id",
"noreply", "source", "holder_id", "holder_id_from", "holder_id_file",
"declared_holder_ids", "declared_holder_ids_file", "holder_ids_refused"}`
(`id` and `noreply` when known; `holder_ids_refused` when the declared ids'
file was refused, which the query reads as declaring none, and says so on
stderr, rather than refusing). It is how a person prints the id to copy into `ABCD_HOLDER_ID`. None of it is pushed. It may
make that one user-endpoint call in the same two cases as a reservation: when
git's identity is a person's but neither its address nor the machine layer
names the holder, and when git's identity is not a person's, whatever the
machine layer says. It takes no record id and no other flag, reserves and
pushes nothing, and writes only the holder id, minted when none is kept and
the variable is not set. It exits 0 when the account and the holder id are
known and 2 when not.

## What it writes

The reservation is `refs/heads/abcd-reserved/<record-id>` on `origin`, a chain
of empty-tree commits written with `git commit-tree` and pushed with one plain
fast-forward push, the repository's hooks running. Each commit's trailers name
the record (`Abcd-Reservation`), the holder (`Abcd-Holder`, the holder id; a
GitHub username on a reservation made before it, still read),
the working copy (`Abcd-Copy`, a random key kept in the checkout's local tier),
the footprint (`Abcd-Footprint`, one per path), the lease (`Abcd-Claimed-At`,
`Abcd-Expires-At`) and abcd's composed `Assisted-by: abcd:<version>` label; a
release adds `Abcd-Released-At`. A drain lane's renewal adds `Abcd-Stage`, the
stage the lane is in, one of `worktree`, `brief`, `implement`, `validate` and
`land`. A fresh or taken-over reservation and a release never carry one, and a
reader reads a tip whose stage is another word, repeated or on a release
commit as unreadable. A release that hands the record back to a
person, which only the implement loop and the drain make, adds
`Abcd-Handed-Back`, the hand-back's kind, and `Abcd-Record`, the blob id of the
record on origin's default branch as last fetched (spc-2610161044461448 step 3,
dcn-2610161040365515). The kind is one of a closed vocabulary: `user-visible`,
`trust-rule`, `design-finding`, `second-package`, `unachievable` (a lane
stopped after its fix rounds), `stuck` and `closed`; the drain writes the last two for a lane stuck
refusing at one stage (spc-2610161044461448 step 5) and for one whose pull
request was closed unmerged (step 8), and every reader reads them. Neither carries free
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
that would take its holder past their cap of live reservations, ten unless
their machine raises it, counted across every record by whose holds are
theirs (the holder id, the declared ids and, on older holds, the login), is refused
naming the cap and the reservations, with the advice to wait for one of their
pull requests to merge, which ends its reservation, rather than release one
whose work is done, or to raise the cap; the refusal is typed, so a drain waits
on it instead of refusing its move, at the same cap
([`35-drain.md`](35-drain.md), spc-2610161044461448 step 6);
a renewal, and a take-over of
the same person's other copy, of a reservation still live add none and are
never refused by the cap, while reviving a lapsed one adds one. A
footprint path that another live reservation's footprint shares, equal or one
a directory holding the other, is refused, unless that reservation is the same
person's from this copy, naming the deeper of the two paths,
the other record, its holder and its expiry. A shared path that is a file
origin's default branch declares generated is no overlap (spc-2610161044461448
step 11), and the act names each such path and the other record
(`generated_shared`, iss-2610282340087862), since whoever lands second
regenerates it after merging. The declaration is `linguist-generated` (set,
or `=true`) in the root
`.gitattributes` of origin's default branch as last fetched, read as stored
with a 256 KiB cap and no fetch; a nested `.gitattributes`,
`.git/info/attributes` and the working tree are never read, because the
declaration must be one every account shares, and a file at or below a
directory holding its own `.gitattributes` is never generated, since that file
could unset the attribute there. Its patterns follow git's
gitattributes rules for the subset abcd reads (a basename at any depth, a
pattern with a slash anchored to the root, `*`, `?`, classes and `**`, a later
line over an earlier one); patterns that set the attribute match
case-sensitively and patterns that unset it match regardless of case, so a
file is generated only where git would call it generated on a clone that
ignores case and on one that does not. A file counts only when origin's default branch
holds it, so a directory is never generated, even when every file under it is.
Every misreading leans towards "not generated": a declaration that cannot be
read, is over the cap, holds a NUL byte or passes the step budget its
matching is held to shares nothing (a line past its own budget for one path
leaves that path not generated) and the act says why
(`generated_unread`, and the back-off's own words), and a line abcd does not
understand (a POSIX class, say) is skipped, voiding every line before it when
it could unset the attribute. abcd shares a generated file only when every line
of the root declaration that git reads (each but a blank or comment line) is
ASCII, with no quoting, macros or checkout conversion (`ident`,
`working-tree-encoding`), and otherwise warns and shares none. A byte-order
mark counts as outside ASCII, even before a comment. A shared path with a
live reservation the same person holds from another copy does not refuse a
person's own act: it is made, and names each such path, the other record, its
expiry and the copy it is held from, a worktree of this repository on this
machine by its path or a copy on another computer, or the person's other copy
with why the worktrees could not be listed when they could not (`own_overlap`,
a warning on stderr in the text form, iss-2610290507526253). Only the
verb asks for this (the request's `WarnOwnOverlap`): a build's and a drain's
reservations are automated and still back off it, as the drain's pass-over
does, so two automated lanes never edit the same files. A renewal the verb makes names
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
raise anyone's ceiling, nor can a `HOME` set inside a working tree that is not
the account's own. A value outside the range, or a key under `reservation`
that nothing reads, refuses an act that adds a reservation (reviving a lapsed
one among them) naming the file, never falling back to ten; no other act reads
the setting, so a bad value never stops a release or the renewal of a live
reservation. The holder's reader (`reservation.github_username`) reads the
same file first for every person whose git email is not a GitHub no-reply
address, so an unknown key under `reservation`, or a file that cannot be read,
stops every act of such a person, a username from their token included. For
anyone else a `config.json` that cannot be read at all (writable by others,
not valid JSON) is read past, ten applies, and the refusal at the cap says
why; at the ceiling of 100 the refusal no longer offers raising it.

The count is by whose holds are the person's (their holder id, the ids they
declare, and their login on older holds), while each machine reads its own
ceiling, so another account whose id is declared here is counted against the
same total under a ceiling no session here can read (iss-2610231004140527). Two places say so. The
refusal at the cap groups the held records by the copy each was made from: this
copy, each other worktree of the repository on this machine by its directory's
name, and elsewhere. And `abcd peers` counts the reader's own live reservations
the same way against this machine's ceiling, warning from four fifths of it
that the total is shared with every session and account whose holds are the
reader's. A sibling
worktree is told apart by the copy key kept in its local tier, read without
minting one and only for the reader's own holds; the key itself is never
shown. The worktrees are listed only by those two readers, never by the view
every other surface shares.

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
branch, and the holder is a self-declared trailer (anyone who reads a holder id
can write it), so the forge's push
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
"release v0.16.0 being cut by h_0123456789abcdef0123456789abcdef (claimed): merges are
held; the hold lapses at <time> unless the cut renews it" (the holder's full
id, marked claimed, as every line naming a release's holder marks each
mention of a holder id that is not the reader's, since no record of pushes
vouches for whose it is; one reserved before the holder id shows `@maya`), judged by the merge
hold's own rule (`claim.JudgedHolds`, iss-2610160415393242): the cutting
copy's own release is shown as "you are cutting release v0.16.0 from this
copy: every other session's merges are held", and a release dated ahead whose
first sighting the checkout cannot keep is shown as not counted, in land's
notice's words, with `own` and `unsighted` on each `release_holds` row of
its machine-readable form. A release reservation that expired with no tag
on origin is a cut its holder stopped renewing: it holds no merge, and
`abcd peers` and `abcd status` name it as lapsed, with its holder, its last
renewal and that from five minutes after the lapse any copy reserves the
release again, to resume the cut or, followed by a release, to withdraw it
(`lapsed_releases`, iss-2610241104098581), so no session waits on a holder
that is gone. The subject is `rel-v` and three plain numbers,
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

### Withdrawn merges

The hold stops a merge from being armed. A merge armed, or queued, before the
release was reserved (a pull request with auto-merge on, or an entry already in
the merge queue) would land when its checks pass, so the release reservation
withdraws it (iss-2610222014550692; dcn-2610222219289525,
dcn-2610222219297651, dcn-2610222219297487, and the round-two rulings
dcn-2610230328220298, dcn-2610230328230097, dcn-2610230328230096,
dcn-2610230328242929). Every forge call goes through the one door to GitHub
(`internal/core/github`), under the session's own `gh` identity, so abcd holds
no token and every withdrawal and re-arm is the acting person's; each GraphQL
field and mutation is pinned in `internal/ghcontract`. Before any withdrawal,
admission or re-arm, the account the `gh` token acts as is compared with the
reservation's holder, and a mismatch touches nothing: a reservation is made
withdrawing nothing, and an admission or a release is refused.

- **One release at a time.** A release reserved anew while another release is
  still being cut backs off, naming it: a second cut would withdraw the merges
  the first admitted. A release is still being cut while its reservation is
  live, and, while its lease has lapsed with pull requests still withdrawn or
  admitted, until its own person's next act ends it five minutes after the
  lapse; the back-off then says how it ends (its holder renews it, or
  releases it). One that lapsed holding
  nothing is abandoned, and blocks nothing. The launch check names the same
  lapsed release, and only that one. A tombstone over it that its holder did
  not push ends nothing (see "When a release ends"), so a second release
  still backs off, naming it. And while the forge's record of pushes cannot
  settle whether a tombstone ends it, the release is read as still being
  cut, and the back-off says why; once this copy has failed to settle it for
  longer than 14 days, a time no writer sets, it is read as ended, so a
  release abandoned while its record could not be read does not hold every
  later one back for good.
- **Withdraw at reserve.** A release reserved anew (every act on it but a
  renewal of this copy's live reservation, so a take-over and a renewal of a
  lapsed one too) pushes its reservation first, so every other session's
  arming is held from then, and then lists the open pull requests into
  origin's default branch that have auto-merge on or sit in the merge queue,
  a hundred a page up to a thousand (more is refused, never cut short). The
  release pull request itself, a same-repository pull request from the
  checkout's branch, is left alone, as is one the reservation already
  admitted. The record comes first: a renewal naming every one about to be
  withdrawn, as withdrawn, is pushed before the forge is touched, so a process
  killed mid-way leaves a record of each merge it may have withdrawn, and one
  it never reached reads as armed when the re-arm comes to it. For each it
  then turns auto-merge off (`disablePullRequestAutoMerge`) and takes it out
  of the queue (`dequeuePullRequest`), the order that leaves the forge no
  arming to queue it again with; a second renewal records what the forge
  actually did when that differs. A renewal from the same copy while live
  keeps the record and withdraws only what it already records as this
  person's, withdrawn or `kept`, that the forge has armed (a process cut off
  between the record and the forge, an arming made by hand since, or a
  withdrawal the forge refused before), read from this person's own newest
  push (see "Whose a withdrawal is"); what another account pushed above it is
  counted and never read.
- **The record.** One `Abcd-Withdrawn` trailer a pull request, at most 40 on a
  tip, six fields separated by one space: its number, its node id, the form it
  was armed in (`queue` for a merge-queue entry, which is also the form of one
  both armed and queued; `auto:merge`, `auto:squash` or `auto:rebase` for
  auto-merge with its method), its head when withdrawn, the holder whose
  reservation withdrew it (their holder id; the GitHub username on a
  withdrawal recorded before the id, which no push of abcd's writes: a
  person's own are written under their kept id in every push of theirs
  that carries them, and read as the ones their login recorded, and an act
  that would write over another holder's still to settle refuses, below),
  and its state: `withdrawn` (still to settle),
  `admitted`, `rearmed`, `armed` (found armed or queued again, counted done),
  `moved`, `closed`, `merged`, `mismatch`, `manual` or `refused` (skipped for
  its owner), or `kept` (the forge refused its withdrawal, so it was left
  armed; this person's own renewals carry it, and a live one tries the
  withdrawal again).
  Every field is fixed-form, so no free text reaches the remote. The reader
  reads it strictly: any other shape, a pull request named twice, more than
  40, one on a record's reservation, or a settled state on a reservation that
  is not released makes the tip unreadable, as a bad hand-back does. Whether a
  withdrawal fits is measured on the tombstone any holder would write (the
  longest holder name, a 39-character username, which is longer than any
  holder id, and the longest label), never the cutter's own. Every commit on the
  release carries forward what its tip still has to settle: this person's own
  reservation keeps its withdrawn, admitted and kept ones, and any other
  commit (a tombstone, a take-over of another person's lapsed one) the
  withdrawn ones.
- **Whose a withdrawal is.** The holder on a withdrawal, and a commit's
  holder, are the writer's data, so every act on a withdrawal (the tag sweep,
  the tombstone sweep, the holder's release, an admission, a renewal's
  withdrawal again) reads a person's withdrawals on a release only from that
  person's own newest push to its branch (dcn-2610231033516293): the newest
  commit the forge's activity record of pushes to the branch (the record
  `abcd status` holds announcements to, C17) shows that person's login pushed
  (the login the holder resolver yields, compared there and never pushed),
  walking the first-parent line from the tip, at most 64 commits. The record
  must place every push it holds on that line, run unbroken down to the push
  that made the branch, and end at the tip read (the activity record passes
  over an entry without a pusher or a commit name, and a record with a push
  missing from it does not say who carried every commit). Commits another
  account pushed above that newest push are never read: they cannot re-arm,
  drop, admit or change the person's withdrawals, and each act names them
  once for each tip, counted by pusher (`not_read`, on the act and in
  `abcd peers`). The person's own entries (their kept id, or their login on
  an entry made before the id) are taken from their newest push, an entry
  under their kept id only when they first recorded it (see "What a copy
  acts on"), and every other entry from the tip, an entry under their kept
  id that they did not first record carried as another's, a declared id's included, whether or not the
  person's own login has pushed to the branch: a take-over between two
  accounts of one person, and its round trip back, carries each account's
  entries as the tip records them, and drops a declared id's entry only for
  the person's own entry for the same pull request. When that newest push cannot be found or
  placed (an app's push above it, which says nothing of which person sent it;
  the record read short, not caught up with the tip, broken or not
  answering; a commit the record places several accounts on; or the push past
  the 64 commits read), abcd does nothing for that person on the release: no
  re-arm, no withdrawal, no admission, no comment. It says so, once for each
  tip on an act and on every `abcd peers` (`unplaced`), and a later act asks
  again; a renewal, an admission or a release over it is refused with nothing
  pushed. A copy that pushes through an app therefore has no newest push of
  its own, and abcd acts on none of its withdrawals: its person re-arms them
  by hand. A renewal or a release over a later commit is judged from the
  holder's own newest push, so a tombstone another login pushed neither ends
  nor renews the release, the admissions are kept, and nothing is withdrawn
  or commented on twice. What the record placed on each commit beneath a tip
  is kept for that tip in the local tier and read by listings alone (`abcd
  peers`, the launch check, the one-release-at-a-time check); every act asks
  the record again, and the local mark of a person with nothing left to do at
  a tip only ever lets an act do nothing, and is kept under the person's
  login and declared ids with their pairings, so a declaration changed
  since reads the tip afresh. The lost walk reads the history
  alone.
- **Admit.** An admission, the reserve verb naming one pull request on the
  release, from the cutting copy's live reservation alone, reads the pull
  request on the forge, records it admitted on a renewal, and only then
  re-arms it in its recorded form, pinned with `expectedHeadOid` to its
  withdrawn head; an arming the forge refuses is put back to withdrawn by a
  second renewal. It is refused from another copy or by another person, on a
  lapsed or released reservation, for a pull request the reservation did not
  withdraw, for one another person withdrew (a release taken over from them:
  the refusal names the remedy, that person taking the release back with
  the take-over flag and admitting it; dcn-2610230833367984), and for one whose
  head moved, or that was closed or merged, since. A release another person
  holds refuses the withdrawer's own admission with the same remedy. Since it
  arms under the cutter's account, it is refused too when the cutter's own
  newest push cannot be found or placed (see "Whose a withdrawal is"), and
  the pull request is admitted only as that newest push records it. Nothing
  is pushed then. An admission of one already admitted reads
  the forge again and arms it when it is not, which finishes an admission cut
  off between its record and its arming. The launch check passes an admitted
  pull request only at the head it was admitted at, and only as the reader's
  own newest push to the branch records it admitted.
- **When a release ends.** A release ends on its holder's release (including
  after a take-over by the same person from another copy), once its tag is on
  origin, or on a tombstone of the holder's that is the holder's own newest
  push to the branch, as the forge's record of pushes says, the holder being
  the login whose own push made the newest reservation on it (another
  person's after a take-over and a release of theirs included). A
  reservation is that login's own when it names them by username or names a
  holder id: the record of pushes names accounts, and no id can be matched
  against one, so an id stands for whoever pushed it, except the reader's
  own ids: the kept id counts only when the reader's own login pushed it, a
  declared id paired with a login only when that login pushed it, and an
  unpaired declared id only when the reader's own login pushed it, so a
  reservation or tombstone naming it that another login pushed is passed
  over and the release is judged by the reservation beneath it. A
  commit another login pushes naming someone else's id can do no more than
  one naming themselves already could. A tombstone its holder did not
  push ends nothing (dcn-2610230833367984): it is named once for its tip
  (`not_ended`), nothing is acted on, and `abcd peers` lists none of its
  withdrawals as due. A lapsed lease ends the cut for its own person alone
  (dcn-2610271926369425, superseding Q1 of dcn-2610230328220298): once
  TakeoverMargin (five minutes) has passed since it lapsed with no tag on
  origin, the next act of any copy of the person whose reservation it is
  claims the end with a tombstone pushed over that lapsed tip, so a renewal
  that lands first wins and the cut keeps its hold, and re-arms only what the
  cut recorded withdrawn, each with one comment saying the cut lapsed. Another
  person's act ends nothing of it. Nothing renews a release lease by itself,
  so a cut still going on renews it; until the margin passes the lapse is
  reported (`abcd peers`, and the launch check refuses the cut).
- **Re-arm.** A settle over a tip that is not yet a tombstone (a tag ended it)
  pushes a tombstone first, with every withdrawal as it stands, as the
  compare-and-swap that claims the end; over a tip that already is one it
  claims nothing again. A second copy of the same person may then settle the
  same release at the same moment and re-arm too, which is harmless: a
  re-arm reads the forge first and is pinned to the withdrawn head, so it
  finds the pull request armed. The cutting copy's release pushes the
  tombstone that ends the reservation, then re-arms every withdrawn,
  unadmitted pull request of its own and checks every admitted one on the
  forge, arming it when it is not, then pushes a second tombstone recording
  each state, only when a state changed, so a refused re-arm writes nothing.
  Once the tag is on origin, any act of a copy whose holder is the
  withdrawal's (every reserve act, a build's and a drain's included, and
  `abcd peers` after its fetch) does the same for that holder's withdrawn
  ones. A re-arm reads the pull request by number and acts only when the
  forge's node id for it is the one recorded (`mismatch` otherwise: node ids
  are global, so a record pairing one pull request's number with another's
  id is never acted on). It uses the form its withdrawer recorded
  (`enablePullRequestAutoMerge` with its method, or `enqueuePullRequest`),
  always with `expectedHeadOid` set to the withdrawn head, so it merges
  exactly the commits its author armed. One already armed or queued counts as
  done, and an admitted one found armed counts as done whatever its head,
  since the forge keeps an arming across pushes; one whose head moved, or that was closed or merged, is skipped and
  named on the tombstone and in `abcd peers` for its owner. One the forge
  refuses because it can merge now and the repository has no merge queue
  ("clean status") settles as `manual`, for its owner to merge: abcd never
  merges another person's pull request. One the forge refused otherwise stays
  withdrawn for the next act, up to five refusals counted by each copy in its
  local tier; after the fifth it settles as `refused`, for its owner. Only the
  forge's own refusal counts: a failure on the way to it (the network, a 5xx,
  `gh` out of time) is tried again without counting.
- **Another person never re-arms.** The re-arm runs under the re-armer's own
  GitHub identity, so a copy of another person carries the withdrawals
  forward untouched and names them, with whose they are (`stranded`), on the
  reservation it makes and in `abcd peers` (`release_withdrawals`).
- **What a copy acts on.** Every act that re-arms, withdraws again or admits
  a merge (a settle, the holder's release, a renewal's withdrawal again, an
  admission) acts on an entry only when it names the person themselves, read
  from their own newest push (their login on an entry made before the id, or
  their kept id when the reservation that first recorded it, held under that
  id, was pushed by their own login), names a declared id paired with a
  login whose own newest push to the branch records that entry exactly and
  that login pushed the reservation that first recorded it under the id, or
  names an unpaired declared id and was first recorded by a reservation held
  under that id that the reader's own login pushed. For an entry the forge
  refused from its first record on (`kept` in every commit that carries
  it), the first record is the lowest of those commits, held to the same
  push. A first record at the last commit of a walk its bound cut short is
  never vouched for, since the commit beneath it is not read. Every other entry is
  carried untouched and listed `stranded`, on the act and in `abcd peers`,
  where `due` lists only what the next act in fact acts on; a stranded entry
  under the reader's own kept id is named as theirs, by why no copy of
  theirs acts on it (their login not known in the session, or the
  reservation that first recorded it not pushed by their own login), never
  as another holder's. For a stranded
  entry naming an unpaired declared id, one line on the terminal
  (`unpaired`, on the act and in `abcd peers`) names the id and
  says that writing its account's login beside it in
  `~/.abcd.noindex/holder-ids` lets this copy settle it; the line names no
  login, and an unpaired id whose entries the reader's own login vouched for
  is never named in it. An admission of a stranded entry of an unpaired id
  is refused with the same line. The
  other-copy back-off over a reservation naming a declared id says, on the
  terminal alone and naming no login, whether the id is paired and, when it
  is, whether the forge's record of pushes says its paired account pushed
  that reservation, so the person can tell one that is not their other
  account's before taking it over. When nothing ties it to an account of
  theirs (the id is unpaired, or its paired account did not push it), the
  back-off ends by saying that `--take-over` is allowed only because the
  declaration lists the id as theirs, that it replaces the reservation and
  ends its hold while acting on none of its withdrawals, and that an id that
  is not theirs is removed from the declaration instead.
- **What ties a push to a login.** The pusher of each commit on a release
  branch is the login the forge's own activity record of pushes to that
  branch names (the repository activity endpoint,
  `repos/<owner>/<name>/activity`, asked for the branch's full ref, the
  record `abcd status` holds announcements to as well), placed on the commits each push carried. Nothing
  on this path falls back to a commit's author or committer fields, which
  any writer can set. Where that record cannot say (a page read short, a
  record not caught up with the tip, a push missing from it, an app's push,
  which names no person, a commit several accounts' pushes are placed on, or
  a push past the 64 commits read), nothing is read as anyone's push and
  nothing is acted on for that person. What a listing reads of the record is
  kept for one tip in the checkout's local tier and read by listings alone;
  an act always asks the forge again. So a take-over carries a paired id's
  entries, and they are acted on, only as far as that check reaches: an
  entry is acted on when the paired login's own newest push records it, and
  that reach is no more than the write access the pushing account already
  holds on the repository.
- **Withdrawals made before the holder id.** A person whose ended release
  holds withdrawals named by their login, and whose home keeps no holder id
  yet, has nothing of them re-armed until their first reservation or
  announcement keeps one: `abcd peers` never mints an id, so its listing says
  each time that held merges from before the holder id stay held until the
  next reservation or announcement (`due_awaits_holder_id`), and a
  reservation then settles them as it keeps the id (an announcement keeps the
  id, and the next listing settles them), its tombstone naming each by the
  kept id. Neither a read nor a push naming the login stands in
  (dcn-2610300525431448). A holder id that is set or kept but refused
  (`ABCD_HOLDER_ID` set to no id, or the kept file refused by its guard) is
  not awaited: every act refuses until it is fixed, so the listing says so
  instead, naming the class of the refusal (`holder_id_refused` on the
  release and at the top of the `abcd peers` and `abcd status` documents,
  with one line on stderr) and never the value or the file's contents.
- **Withdrawals another holder made before the holder id.** An entry naming
  another holder by their login names that login, which nothing abcd pushes
  carries, so an act that would write one still to settle (withdrawn,
  admitted or kept) into a new commit, a take-over or a settle among them,
  refuses before it pushes anything or touches the forge, with one line
  saying how many the release holds and that the holder's own copy settles
  them first; it names no login. One of theirs already settled needs nothing
  more and is not written.
- **Lost withdrawals.** An older abcd ignores the `Abcd-Withdrawn` trailer, so
  its renewal, take-over or tombstone rewrites the tip without them. For each
  release claimed in the last 14 days, `abcd peers` and the launch check walk
  the branch's history and name each withdrawal still withdrawn on an ancestor
  that the commit above it and the tip do not carry (`lost_withdrawals`, the
  `merge-hold-lost` warning): nothing re-arms those, so their withdrawer does
  it by hand. abcd's own correction after a refused withdrawal keeps the
  entry (`kept`), so it never reads as lost.
- **A comment on each pull request acted on** (dcn-2610230712084099,
  superseding dcn-2610230328230097; the sentences dcn-2610230833367984). abcd
  posts one fixed sentence naming the release's tag, through `gh`'s
  `addComment` (pinned in ghcontract's `graphql-fields.txt`), on a pull
  request it withdraws whole ("the merge of #N is withdrawn by HOLDER while
  release vX is cut, and is re-armed after it."), admits ("is admitted into
  release vX by HOLDER.", the admitter, who is the withdrawer; HOLDER is
  their holder id, and the comment's author, as on any comment, is the
  account gh posts it as), re-arms ("is
  re-armed; the hold for release vX has ended."), finds armed already ("is
  armed; the hold for release vX has ended.") or leaves for its owner ("is not
  re-armed; the hold for release vX has ended." and one clause from a closed
  set: head moved, no merge queue, refused five times, closed, merged), each
  prefixed "abcd release hold: ". Every real disarm is posted, a live
  renewal's withdrawal again included; nothing is posted for a refused (`kept`)
  or partial withdrawal, a `mismatch`, any other state, an admission run
  again, or a second copy whose settle does not land. Every value in it is
  validated data, the holder carries no at-sign, so it mentions no one (every
  subscriber of the pull request is notified, as by any comment), and the body
  is judged by the outbound policy before it is posted, a refused one never
  posted. The comment is read back from the forge by the id it answered with;
  one that reads back as other than what was sent is reported, never edited.
  It is posted only after the push recording that state lands. A comment that
  fails is a `comment` failure, named by number and reason, and never holds up
  or undoes the act it reports.
- **Failures are loud and lose nothing.** A withdrawal the forge refuses
  (no permission, no network) leaves the reservation standing and is named
  by number and reason, and the second renewal records it `kept`, which the
  next live renewal tries again; a
  partial one (auto-merge off, the queue entry kept) is recorded as what
  happened, in the form it lost. A pull request the record has no room for is
  not withdrawn, and is named. If the renewal recording the withdrawals
  cannot be pushed, the forge is not touched. The launch check
  (`04-launch.md`) refuses a cut while any merge the reservation does not
  account for is armed, which catches each of these and a merge a person
  arms by hand on the forge after the reservation; the cutting session still
  tells people working in the repository by hand to hold.

## Exits

Exit 0 when the act reached origin, or a release was already made. Exit 3, the
back-off, when another person's reservation is live or lapsed under the
margin, an unreadable one is within its hold, another push won the race, or the
same person's other copy holds it and no take-over was asked for, or a
footprint path that is not a generated file is in another person's live
reservation's footprint (or, for a build's or a drain's reservation, the same
person's from another copy), or a drain's reservation meets a hand-back that stands,
or the record is finished (resolved, won't-fixed, shipped or superseded) on
origin's default branch. Exit 2 on the cap,
on a fork origin, and on every other refusal and failure. The JSON document carries `act`, `record`, `holder`,
`copy` (`this` or `another`, never a copy's key), `footprint`, `claimed_at`,
`effective_expiry`, `commit`, `replaced` for a take-over,
`derived_unchecked` when the default branch could not be read,
`directory_footprint` (`path`, `tracked_files`, `on`) for each footprint path
naming a directory,
`fork_unchecked` when the forge could not say whether origin is a fork,
`fork_allowed`, the fork's owner/name, when the check was waived for the fork
the repository allows, and
`handed_back` (`kind`, `holder`, `at`) for a reservation made over a hand-back
that stands, which the text form prints as a `note:` line, `hand_back_kept`
(the same three) for a release whose tombstone kept that hand-back, also a
`note:` line, and `hand_back_unpublished` for a release whose tombstone marks
no hand-back it would otherwise have marked, a `no hand-back:` line,
`own_overlap` (`path`, `record`, `effective_expiry`, `copy`, `copy_unread`)
for each path shared with the same person's reservation from another copy, a
`warning:` line on stderr, `dropped_footprint` for each path the person's own replaced
reservation held that the footprint leaves out, a `warning:` line on stderr,
and `generated_unread` when generated files were not recognised, a `not
recognised:` line. A release's acts add `withdrew` and `withdrawals` (each
`number`, `id`, `form`, `head`, `by`, `state`), `admitted` for an admission
(`act: admitted`), `stranded`, `unpaired` (each unpaired declared id among
the stranded), `settles` (`subject`, `commit`, `settled`,
`failures`, `failed`), `forge_failures` (`number`, `act`, `reason`) and
`forge_unchecked`; the text form warns of every failure on stderr. An
admission refused exits 2.

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
| `--admit` | int |
| `--apply` | bool |
| `--footprint` | stringArray |
| `--holder` | bool |
| `--lease` | duration |
| `--older-than` | int |
| `--release` | bool |
| `--sweep` | bool |
| `--take-over` | bool |

<!-- surface-appendix:end -->
