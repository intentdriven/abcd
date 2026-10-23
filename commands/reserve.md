---
name: reserve
description: "Reserve an issue, intent or release on the shared remote, or release it: Writes one commit to its origin branch; refuses what someone else holds, with exit 3."
argument-hint: "<record-id> --footprint <path> [--footprint <path>…] [--take-over] [--lease <duration>] | rel-vX.Y.Z [--lease <duration>] | rel-vX.Y.Z --admit <pr> | <record-id|rel-vX.Y.Z> --release | --holder"
block: agents
user-invocable: false
---

# `/abcd:reserve`

Reserve an issue or an intent on the shared remote before starting work on it,
so every other participant (a second person, a second computer, a hosted cloud
session) sees that it is taken before starting the same item. This verb is an
agent's, and so is `/abcd:peers`, which lists the reservations: a person sees
them on the board (bare `/abcd:abcd`) and on a record's page
(`/abcd:abcd <record-id>`), not by typing either.

**What it does to the remote.** The command's documented meaning is a push: it
writes one commit to the branch `abcd-reserved/<record-id>` on the remote named
`origin`, and to nothing else. The commit changes no file (its tree is the
empty tree); its trailers name the record, the holder, this working copy, the
footprint and the lease. The push is a plain fast-forward with the
repository's hooks running, never forced and never with `--no-verify`, and it
touches no HEAD, index or working tree here. When git's identity does not name
the holder (below), it also makes one read-only call to GitHub's user endpoint,
through `gh`, to learn whose account the forge token `gh` resolves from this session's
environment belongs to. A release reservation also reads and writes the
pull requests' merge state on GitHub, through `gh` under the session's own
identity (below): it withdraws the merges armed on the default branch, and
re-arms them when the release ends (its `--release`, or its tag on `origin`;
a lapsed lease ends nothing).

## Reserve before starting

Run, naming every repository-relative path the work will change, once per path:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve iss-2609020716570699 --footprint internal/core/claim/claim.go --footprint commands/reserve.md --json
```

The record is an issue (`iss-N`) or an intent (`itd-N`). The footprint is
required. `--lease <duration>` sets how long the reservation holds, from `1m`
to `24h`; the default is two hours. Running the same command again from the
same copy renews it.

The payload carries `act` (`reserved`, `renewed`, `taken_over`, `released` or
`already_released`), `record`, `holder`, `copy` (`this` or `another`, never a
copy's key), `footprint`, `claimed_at`, `effective_expiry`, `commit`, and
`replaced` when a take-over replaced a reservation: its `holder`, `copy`,
`commit`, `effective_expiry`, and `unreadable` when it could not be read. When
origin's default branch could not be read, `derived_unchecked` says why: the
act went on, but whether the record is already finished there was not checked.
When GitHub could not say whether origin is a fork (gh missing, out of time or
refused), origin's host is not github.com by name, a push URL could not be
read, or a github.com URL does not name exactly one owner and one repository,
`fork_unchecked` says why: the act went on, and the text reads `not checked: whether origin is a fork
(…)`. When origin is the one fork the repository allows, `fork_allowed` names
it as `<owner>/<name>` and the text reads `origin is a fork the repository
allows (<owner>/<name>)`. Pass either on to the user.
`own_overlap` lists each path the reservation shares with the user's own from
another copy, as `path`, `record` and `effective_expiry` (see **When it
refuses**); the text form warns of each on stderr. `generated_unread` says why
no generated file was recognised, so every shared path was judged an overlap.
Pass both on to the user.

Tell the user what was reserved and until when.

## Reserve a release before cutting it

A session cutting a release reserves the release itself, so every other
session's land step, the implement loop's and `abcd land`, holds its merge
until the cut is done:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve rel-v0.16.0 --json
```

The subject is `rel-v` and the version, three plain numbers (`rel-v0.16.0`,
never `rel-0.16.0` or `rel-v0.16.0-rc1`). It takes no footprint (one is
refused, exit 2), and it does not count toward the cap. With no
`--lease` it holds for `24h`, so a cut that waits on checks and an approval
outlasts it; running the same command again from the same copy renews it for
another day, and without a renewal it holds no longer than a day (see
**Clocks** for one claimed ahead of this copy's clock). It ends by itself once the tag
`v0.16.0` is on `origin` (the listing shows it `released`, `released_by:
tag`), or with `--release` from the copy that holds it. Reserving a release
already tagged on `origin` is refused (exit 2). While it is live, an
`implement step` at the land fence or an `abcd land` in any other copy, the
user's own other computer included, exits 3 naming the release and its holder
and arms nothing;
this copy's own landings are not held. It also withdraws every merge armed or
queued before it (**Withdrawn merges** below). The hold binds sessions, not
people, so the cutting session tells people working in the repository by hand
to hold.
`/abcd:peers` shows the line "release
v0.16.0 being cut by @octo: merges are held", by the same rule the hold
counts with: in the cutting copy it shows "you are cutting release v0.16.0
from this copy: every other session's merges are held", and a release dated
more than five minutes ahead whose first sighting this checkout cannot keep
is shown as not counted, in the words `abcd land`'s notice uses; `peers
--json`'s `release_holds` rows say so with `own` and `unsighted`. The fetch every act makes reads
`origin`'s release tags with the reservations, into `refs/abcd/release-tags/`,
and never touches this checkout's own tags. Those are ordinary refs in this
checkout, so `git log --all` lists the commits they name and `git push
--mirror` would push them.

## Withdrawn merges: admit and re-arm

The hold stops a merge from being armed, so a release reserved anew (any act
but a renewal of this copy's live reservation) also withdraws every merge
already armed: after its own push holds every other session's arming, it lists
the open pull requests into `origin`'s default branch with auto-merge on or in
the merge queue, records each on a renewal of the reservation, with the form it
was armed in (`queue`, or `auto:merge`, `auto:squash`, `auto:rebase`) and its
head, and only then turns each one's auto-merge off and takes it out of the
queue. The release pull request itself (one from this checkout's branch, in
this repository) is never withdrawn. The payload's `withdrew` lists them, as
`number`, `id`, `form`, `head`, `by` and `state`, and `withdrawals`
everything the reservation records. A renewal from the same copy while the
reservation is live keeps the record and withdraws nothing new: it withdraws
again only a pull request it records as yours, withdrawn, that the forge has
armed again (a run cut off between its record and the forge, or an arming made
by hand since), and only one the branch's history and the forge's record of
pushes vouch for, as they vouch for a re-arm (below); any other is named in
`settles` and left as it is. A pull request the forge refused to withdraw stays
on the record as `kept`, carried by your renewals, and the next live renewal
tries the withdrawal again, so reserving the release again is the remedy. Tell
the user what was withdrawn.

One release is cut at a time: reserving a release while another release is
still being cut backs off (exit 3), naming the release and its holder. A
release is still being cut while its reservation is live, and also once its
lease has lapsed while it still holds pull requests withdrawn or admitted; the
back-off then says how it ends: its holder renews it, or ends it with
`--release`. A lapsed release that holds nothing does not block.

Every withdrawal and re-arm is made through `gh` under the account its token
acts as, so each is refused when that account is not the reservation's holder:
the reservation stands, nothing is withdrawn, and `forge_unchecked` says which
account `gh` acts as; `--admit` and `--release` are refused (exit 2, nothing
pushed). Switch `gh` to the holder's account and run it again. When `gh`
cannot say which account it acts as at all (not logged in, no network), a
reservation withdraws nothing and says so in `forge_unchecked`, `--admit` is
refused (exit 2, nothing pushed), and `--release` still pushes the tombstone
but re-arms nothing, naming why in its settle's `failed`; the next act of a
copy of yours that fetches the reservations re-arms them once `gh` answers.

To let a withdrawn pull request into the cut, from the copy that holds the
release:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve rel-v0.16.0 --admit 1234 --json
```

It renews the reservation recording that one admitted, then re-arms it in its
recorded form, pinned to its withdrawn head (`act: admitted`, `admitted`). It
is refused (exit 2, nothing pushed) from any other copy, by any other person,
on a lapsed or released reservation, for a pull request the reservation did
not withdraw, for one another person withdrew (a release you took over: its
withdrawer admits it, or takes the release over themselves), and for one whose
head moved, or that was closed or merged, since. It is also refused, since an admission arms the pull request under your
account, unless the forge's record of pushes shows you pushed the commit that
first recorded the withdrawal: when no reservation of yours recorded it first,
or another account pushed the commit that did; when an app pushed that commit
(arm it by hand if it belongs in the cut); and when the record does not settle
it yet (run `--admit` again in a minute, once the record has caught up). Admitting one already admitted reads the forge again and arms it when
it is not armed, so an `--admit` cut off between its record and its arming is
finished by running it again (the release's settle arms it too). When the
forge then refuses the arming, a second renewal puts it back to withdrawn and
the refusal says so. The launch check lets an admitted pull request through
only at the head it was admitted at, and only once the forge's record of
pushes shows you pushed the reservation and the admission.

Every other withdrawn pull request is re-armed when the release ends, and an
admitted one is checked on the forge and armed if it is not. A release ends
in any of these ways: `--release` from the cutting copy; its tag on `origin`;
or a tombstone on its branch that the holder of the reservation it ends
pushed, as the forge's record of pushes says, including another person who
took the reservation over and released it. A tombstone its holder did not
push ends nothing: it is named once for its tip (a settle's `not_ended`, and
`/abcd:peers`), and nothing is re-armed until the holder's `--release` or the
tag. `--release` pushes the tombstone
first, with every withdrawal as it stands, then re-arms; once the tag is on
`origin`, the next act of any copy of the same person that fetches the
reservations (any reserve, a build or drain start, `/abcd:peers`) does the
same, pushing a tombstone before it touches the forge when the tip is not one
yet. Two copies of the same person that settle the same release at once may
both re-arm; that is harmless, because a re-arm reads the forge first, is
pinned to the withdrawn head, and finds the pull request already armed. A
tombstone recording the new states is pushed only when a state changed, so a
re-arm the forge keeps refusing writes nothing to `origin` while it is tried
again. **A lapsed lease ends nothing.** Nothing renews a release lease by itself, so a slow cut's lease
lapses while it is still being cut: the lapse is reported (`/abcd:peers`, and
the launch check refuses the cut until it is renewed), and no copy re-arms
anything. Renew it by reserving the release again from the cutting copy.

**After a crash.** When the copy cutting the release is gone (its worktree
deleted, its machine down), end the release from another copy of the same
person: `reserve rel-vX.Y.Z --take-over`, which carries the withdrawals, then
`reserve rel-vX.Y.Z --release`, which re-arms them.

Each is re-armed in the form its withdrawer recorded, pinned to the head it
had: the form is part of the record the forge's record of pushes vouches for,
so an entry whose form a later commit changed is that commit's record, not the
withdrawer's, and is acted on only when the withdrawer pushed that commit. One
already armed or queued again counts as done (`armed`); an admitted one found
armed counts as done whatever its head, since the forge keeps an arming across
pushes. One is skipped and
named for its owner to act on: its head moved (`moved`), it was closed or
merged (`closed`, `merged`), the node id recorded for it is not the forge's for
its number (`mismatch`, never acted on), or it can merge now and the repository
has no merge queue to put it in (`manual`: abcd never merges another person's
pull request, so its owner merges it). The tombstone records every state, so no
copy acts twice; one the forge refused stays `withdrawn`, and the next such act
tries it again, up to five refusals counted by each copy; after the fifth it
is recorded `refused` and left to its owner. Only the forge refusing the arming
counts: a failure on the way to it (the network, a 5xx, `gh` out of time) is
named and tried again without counting. The payload's `settles` names
each release settled this way, with its `settled` pull requests, its
`failures` and the tombstone's `commit`.
Only a withdrawal whose first record is a reservation of its withdrawer's on
the branch's own history, and whose commit the forge's record of pushes says
that person pushed, is ever acted on: re-armed, admitted, or withdrawn again.
One that is not is left alone and named once for each tip: `unrecorded` (no
reservation of theirs recorded it first, or another account pushed the commit
that did) and `app_pushed` (an app pushed that commit, which says nothing of
which person sent it: re-arm it by hand). Either verdict is kept for the tip in
the local tier, so the forge is asked once, not on every act. One the forge's
record does not settle yet (read short, not caught up with the tip, not
answering, or with a push missing from it) is named `unverified`, with
`unverified_why`, and a later act asks again. A copy that pushes through an app
(a hosted session's installation token, say) can therefore neither admit nor
re-arm its own withdrawals: every one is `app_pushed`, and the person re-arms
each by hand after the cut. A change of a withdrawal's state counts only when
its withdrawer pushed it too: one the tip records settled (`kept`, re-armed,
skipped) where the commit that moved it out of withdrawn or admitted is not
one the forge's record says its withdrawer pushed is named once for each tip
(`restated`) and acted on as it stood before that commit, so the release
re-arms it, and a reservation made over such a tombstone carries it. A copy of another person
never re-arms: a reservation it makes over the release carries the
withdrawals forward untouched (`stranded`, and a `note:` line naming whose
they are), and `/abcd:peers` names them (`release_withdrawals`). A
withdrawal recorded on an earlier commit of the branch and missing from a
later one, with the tip not carrying it either, is reported lost
(`lost_withdrawals` on `/abcd:peers`, and a warning in the launch check): an
older abcd that renews the release drops the record that way. abcd's own
correction after a withdrawal the forge refused keeps that entry (`kept`)
instead, so it is never reported lost.

**A comment on each pull request it acts on.** abcd posts one fixed sentence
on a pull request, naming the release's tag, when it does one of these, once
the push recording it has landed:

- withdraws it whole (auto-merge off, and out of the queue): "abcd release
  hold: the merge of #1234 is withdrawn by octo while release v0.16.0 is cut,
  and is re-armed after it." Every such disarm is posted, a live renewal's
  withdrawal again included (after a run cut off before the forge, or after
  its owner re-armed it by hand);
- admits it: "abcd release hold: the merge of #1234 is admitted into release
  v0.16.0 by octo.";
- re-arms it: "abcd release hold: the merge of #1234 is re-armed; the hold for
  release v0.16.0 has ended.", or finds it armed already: "abcd release hold:
  the merge of #1234 is armed; the hold for release v0.16.0 has ended.";
- leaves it for its owner: "abcd release hold: the merge of #1234 is not
  re-armed; the hold for release v0.16.0 has ended." and one clause saying
  why: its head changed while it was held (`moved`), it can merge now and
  there is no merge queue (`manual`), GitHub refused to re-arm it five times
  (`refused`), or it was closed or merged.

Nothing is posted for a withdrawal the forge refused (`kept`), a partial one
(auto-merge off but still queued, which `forge_failures` names), a node id
that is not the forge's for its number (`mismatch`), an admission run again,
or a second copy whose settle does not land; a re-arm whose tombstone was not
pushed is posted by the act that next records it. The sentence carries
nothing but validated fields: no session URL, no tool footer, and the login
without an at-sign, so it mentions no one (every subscriber of the pull
request is notified, as by any comment). Each one is judged by the outbound
policy the commit-msg hook judges a commit message by before it is posted,
and a body the policy refuses is never posted; once posted it is read back
from the forge, and one that reads back as other than what was sent (a footer
appended, say) is reported for you to correct on the pull request, never
edited. A comment that fails is named in `forge_failures` (or a settle's
`failures`) with act `comment`, and never holds up or undoes the withdrawal
or the re-arm it reports.

**Failures are loud and lose nothing.** A withdrawal or re-arm the forge
refuses is named in `forge_failures` (`number`, `act`, `reason`) and on
stderr, and the reservation stands; a merge left armed is what the launch
check refuses the cut on (`/abcd:launch`). `forge_unchecked` says why the
merges could not be listed or touched at all: `origin` is not one repository
on github.com, the forge did not answer, or `gh` acts as another account. A
pull request the 4096-byte message (at most 40) could not record is not
withdrawn, and is named. When the renewal recording the withdrawals cannot be
pushed, the forge is not touched and each one stays armed, named. Pass every
one of these on to the user.

## When it refuses

**Exit 3 means back off: take other work.** The record is reserved by someone
else and still live, or lapsed less than five minutes ago, or its reservation
cannot be read and is still within its two-hour hold, or another push reserved
it while this one was on its way. Say who holds it and until when, and pick
something else. Never retry in a loop.

**Exit 3 also covers a shared path.** A footprint path that another person's
live reservation also holds, for whatever record, is refused, naming that path,
the other record, its holder and until when. Two paths are shared when they are
equal or one is a directory holding the other. A shared path that is a file
`origin`'s default branch declares generated is no overlap: generated files are
those its root `.gitattributes`, as last fetched, marks `linguist-generated`
(or `linguist-generated=true`). That one file is read, never a nested
`.gitattributes`, `.git/info/attributes` or this checkout's own copy, because
the declaration must be one every account shares; reading it is a disk read,
never a fetch. Patterns that set the attribute match case-sensitively;
patterns that unset it match regardless of case, so on a clone that ignores
case nothing is shared that git there would not call generated. A file at or
below a directory that holds its own
`.gitattributes` is never shared, since that file could unset the attribute
there. A directory is never generated, even when every file under it
is, so two directory footprints that share still overlap. When the file is
there but cannot be read (it is not a regular file, is over 256 KiB, holds a
NUL byte, or takes too long to match against the files),
nothing is shared, the refusal adds "generated files were not recognised: …",
and a reservation that is made says the same (`generated_unread`). abcd
shares a generated file only when every line of the root declaration that git
reads (each but a blank or comment line) is ASCII, with no quoting, macros or
checkout conversion (`ident`, `working-tree-encoding`), and otherwise warns
and shares none. A byte-order mark counts as outside ASCII, even before a
comment.
Running the command again to renew is checked the same way, since it names
its footprint anew, so a reservation is never widened onto another; a build's
or a drain's own renewal keeps its footprint and is not checked while the
reservation is live, but is when it revives a lapsed one. Take other
work, or reserve with a footprint that does not overlap. The cap below and this
overlap are judged on the reservations fetched just before the push, and each
reservation is its own branch, so two people reserving at the same moment can
both pass them: they are best-effort under a race. Two reservations of the
same record cannot both land; the second push is refused.

**A path shared with your own other copy warns, and the reservation is
made.** The user's own reservations from this copy are no overlap at all,
since that work is this copy's. One the user holds from another computer or
another checkout that shares a path with this reservation does not refuse it:
the act goes ahead, exit 0, and warns on stderr, `abcd reserve: warning:
"<path>" is also in the footprint of <record>, which you hold from your other
copy until <time>; reserved anyway, since both are yours` (`own_overlap` in
`--json`). Tell the user both pieces of work change that path. Only this verb
does so, a reserve with `--take-over` included. A build's and a drain's
reservations are automated and still back off such a path, and so does a
drain's pass-over.

**Exit 3 also covers your own other copy.** When the same person holds the
record from another computer or another checkout, the refusal says "you are
already building … on another computer". Pass `--take-over` only when the user
confirms the work moves here; it moves only the user's own reservation, and the
output names the copy it replaced. It never moves another person's.

**Exit 2 when the user is at the cap.** One person may hold at most ten live
reservations by default, counted across every record by username, whichever of
their sessions and copies holds them. A reservation past that is refused,
naming the cap and the reservations; a renewal, and a take-over of the user's
own other copy, of a reservation still live are never refused by the cap, while
renewing one that has lapsed counts as a new one. The refusal advises waiting
for one of the user's pull requests to merge, which ends its reservation: most
of them are usually lanes whose pull requests wait on a merge, and releasing
one would free an issue whose work is done. Release one only when its work is
abandoned, if the user agrees. A person who runs many sessions at once raises
the cap with `reservation.max_live` in their own `~/.abcd.noindex/config.json`
(a whole number from 1 to 100), never in a repository's `.abcd/config.json`,
which is not read for it; a value outside that range, or a misspelt key beside
it, is refused naming the file.

**Exit 2 on a fork origin.** A new reservation or a take-over asks GitHub
whether each URL origin pushes to is a fork; a renewal and a release never ask,
except a drain's reservation that renews your lapsed one, which is the drain's
hold afresh and asks as a new one does.
A fork is refused before anything is pushed, since a reservation there is
invisible to everyone working on the repository it was made from. The refusal
names that repository and the `git remote set-url` command that points origin
there; relay it, and run the command only if the user agrees. When the fork is
itself where everyone working on the repository reserves, the repository says
so by setting `reservation.fork_origin` to that fork's `"<owner>/<name>"` in
`.abcd/config.json` (the default is `"refuse"`); never set it to get past the
refusal without the user's word. It allows that one fork only: a fork of it is
refused naming both, and a value that is neither `"refuse"` nor one owner and
one name is refused, naming the file, once origin is a fork.

**Exit 2 is every other refusal**: an id that is not an issue, an intent or a
release (`rel-vX.Y.Z`), no footprint for a record or one for a release, a
release already tagged on `origin`, a path that is not repository-relative, a lease out of bounds, a
message over the 4096-byte cap, no GitHub account (the token could not be
asked or did not answer), no local tier (`.abcd/.work.local/`, which `/abcd:ahoy install` creates), no remote
named `origin`, a fetch or a push origin refused. Each refusal ends by saying
nothing was pushed or no reservation was made, and quotes git's own reason with
the home directory redacted. Relay it.

**What a reservation is not.** Reservations coordinate people who share push
access to the repository and act in good faith. They are not an access
control: anyone who can push can append a commit to another holder's
reservation branch, and the holder a reservation names is read from its own
trailer, set from that writer's `user.email`, machine configuration or token,
so it is self-declared. The forge's push permissions are the only boundary.

**Clocks.** A reservation's expiry is written by the holder's clock and
judged by the reader's, and the five-minute margin is all the disagreement it
absorbs. A holder whose clock runs behind loses that much of each lease: one
behind by more than half the lease plus the margin (65 minutes for the
default two hours) can be taken over between its own renewals, and a
reservation made once and never renewed by a clock behind by more than its
lease is open to a take-over as soon as another copy fetches it. A
reservation dated more than five minutes ahead of the reader counts from when
that copy first fetched it, so it holds at most a day from then. Keep the
machines that reserve on network time.

**The holder** is the user's GitHub username, from the first of these that
names it: `git config user.email` when it is a GitHub no-reply address;
`reservation.github_username` in `~/.abcd.noindex/config.json`; and the
account the session's own GitHub token belongs to, through one call to
GitHub's user endpoint. The repository's own configuration is never read for
it. When git's identity is not a person's (unset, or a machine's, as in a
hosted cloud session), the token alone names the holder and that account
authors the reservation commit; the identity git had is shown nowhere. When
nothing names the holder the refusal says why; relay it.

To learn the holder without reserving anything:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve --holder --json
```

It prints `{"login", "id", "noreply", "source"}` (`source` is `git`, `config`
or `token`; `id` and `noreply` when known), or as text `would reserve as
<login> (from <source>)`. It may make that one user-endpoint call in two
cases: git's identity is a person's but carries neither a no-reply address nor
a machine key; or git's identity is not a person's (unset, or a machine's), when
the machine key is skipped. It takes no record id and no other flag, and
reserves, writes and pushes nothing. Exit 0 when the account is known, 2 when
not. Its `noreply` is the account's address; a person's own git identity still
authors their commits.

## Release when done

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reserve iss-2609020716570699 --release --json
```

A release pushes a tombstone child and deletes nothing. It takes no footprint,
take-over or lease. Only the holder's own reservation from this copy is
released (exit 2 otherwise); a release already made exits 0 as
`already_released` and pushes nothing. A record that is resolved, won't-fixed,
shipped or superseded on origin's default branch counts as released whatever
its reservation says, and so does a release whose tag is on `origin`. Such a
record is not reserved again: a reservation or take-over of
it backs off (exit 3) saying it is finished on origin's default branch, with a
tip or without one, and pushes nothing.

**Over a hand-back.** A drain that hands an issue back to a person releases
its reservation with a tombstone naming the hand-back's kind (one of
`user-visible`, `trust-rule`, `design-finding`, `second-package`,
`unachievable`, `stuck`, `closed`; the drain writes `stuck` for a lane that
kept refusing at one stage and `closed` for one whose pull request was closed
unmerged) and
the blob id of the issue's record on
origin's default branch. Every drain backs off that issue until a person
changes the record there, and no drain takes over or renews a reservation you
make over it, lapsed or not. Your own reservation over it is made, and the
result says whose hand-back it was made over, of which kind and when
(`  note: ...` in the text form, `handed_back` in `--json`): read the record's
hand-back before you start, since it names a decision that is a person's.
Releasing it again without changing the record keeps the hand-back: the
release's tombstone carries its kind and blob forward, and the release says so
(`  note: ...`, `hand_back_kept` in `--json`), so no account's drain takes the
issue until a person changes the record. A release whose tombstone cannot mark
the hand-back (the record is in no one file on origin's default branch, that
branch could not be read, or the message would pass its cap) is still made, plain, and says why on a
`  no hand-back: ...` line (`hand_back_unpublished` in `--json`).

**Reopening a record.** The back-off reads origin's default branch alone, so a
record being reopened on a branch (moved from `resolved/` or `wontfix/` back to
`open/`, or an intent from `shipped/` or `superseded/`) still backs off, and no
flag, `--take-over` included, gets past it. Merge the reopen first: once
origin's default branch holds the record open again, reserve it
(dcn-2610161135122294). Tell the user that, rather than retrying.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
