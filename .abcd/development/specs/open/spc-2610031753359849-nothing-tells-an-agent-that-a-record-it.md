---
id: spc-2610031753359849
slug: nothing-tells-an-agent-that-a-record-it
intent: itd-2609091034175565
origin: researcher-authored
production_mode: hand-written
---
# Reservations on the shared remote: `abcd reserve`, one liveness judgement, and the build, drain and land integration

## Summary

This spec delivers
[itd-2609091034175565](../../intents/planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md):
Before an agent starts on an item, abcd reserves it on the shared remote as a
branch nobody merges, and every participant (a second person, a second computer,
a hosted cloud session) sees the reservation before starting the same item or
touching the same files.

A reservation is `refs/heads/abcd-reserved/<record-id>` on `origin`: a chain of
empty-tree commits written with `git commit-tree`, whose trailers name the
record, the holder (a GitHub username), the working copy, the footprint and the
lease. Every act is one plain fast-forward push. One new core package,
`internal/core/claim`, owns the format, the holder, the copy key, the fetch, the
push and the view; one new leaf package, `internal/core/lease`, holds the
liveness judgement that the shipped in-run `implement claim` and the reservation
both call. The front doors are a new top-level verb, `abcd reserve`, wired on
the CLI and on the plugin markdown surface, and the existing `abcd peers`,
`abcd build`, `abcd drain`, `abcd implement step`, `abcd <record-id>` and the
board.

The rule enacted is
[adr-2610031352212697](../../decisions/adrs/2610031352212697-claims-are-branches-on-the-shared-remote.md)
(proposed; it moves to accepted when spec 2 closes). The run lessons and the
state of the art behind it are in
[`2026-10-03-multi-agent-coordination-sota-and-run-lessons.md`](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md)
and are not repeated here.

**Two specs, one record** (intent decisions 19 and 38). This spec lists the
spec-1 steps first (reserve, renew, release, the listing and the pre-push hook)
and the spec-2 steps after them (the land fence, the cap, the footprint
refusal, the renders, and the build and drain integration). When the spec-1
steps have landed, `abcd spec close spc-2610031753359849 --remainder <slug>`
mints spec 2 carrying the unlanded steps, and the intent ships when spec 2
closes. Steps 1 to 7 below are spec 1 and steps 8 to 15 are spec 2, each
title marked with its spec.

**On words.** Every surface a person reads says **reservation**; inside the
code a reservation is a **claim**, the name the shipped in-run claim already
uses (the intent's "On words").

## Scope

In, as the intent's
[What's In Scope](../../intents/planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md#whats-in-scope)
draws it:

- **Spec 1.** `internal/core/lease` (new: the lease bounds and the one
  liveness judgement, extracted from `internal/core/implement/claim.go`);
  `internal/core/claim` (new: the message format and its reader, the holder,
  the working-copy key, the fetch, the push protocol, the derived release and
  the view); `internal/core/implement` (the in-run claim calls the shared
  judgement); `internal/gitutil` (the networked git runner lifted out of
  `internal/core/implement/loop/land.go`, and the empty-tree name); the
  `reserve` verb and the reservation listing in `abcd peers`
  (`internal/surface/cli`); `.githooks/pre-push`; a reservation-push
  `known_good` in `internal/core/guard/defaults/guard.json`; the plugin pages
  `commands/reserve.md` and `commands/peers.md`; the generated command
  reference.
- **Spec 2.** The cap and the footprint overlap refusal (`internal/core/claim`
  and `abcd reserve`); the run's reservation in `internal/core/implement/loop`
  (taken at `start`, renewed by `advance`, fenced in `landArm`, tidied in
  `landMerged`); the exclusions in `build next` and the `peers` check in
  `build <itd-N>` (`internal/core/implement/loop/check.go`, `next.go`); the
  render in `abcd <record-id>` (`internal/core/record`) and on the board
  (`loop.StatusPeers`, `internal/surface/cli/board_status.go`); the `--release`
  and `--take-over` flags on `abcd build`; the drain's reservation per issue;
  the plugin pages `commands/build.md`, `commands/drain.md`,
  `commands/implement.md` and `commands/abcd.md`.

Out, as the intent's
[What's Out of Scope](../../intents/planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md#whats-out-of-scope)
draws it: The wider session register (itd-2609150819440345); outside
contributors (itd-2610031259176838); the resolve-time refusal
(itd-2609091416304128), which reuses this spec's "as of last fetch" reader when
it is built; deleting reservation branches (itd-118's sweep); keeping one
person's sessions in one copy apart (the in-run claim, unchanged); judging two
records as one observation (itd-87); worktrees (itd-2609091014076309); any field
on a record file; a second backend; a lock; detecting whether a session is
alive; any implicit network traffic; a remote not named `origin`; and any change
to the user's git configuration.

## Approach

### The reservation commit

A reservation commit is written by `git commit-tree <empty-tree> [-p <parent>]
-F -`, with the message on standard input, from the checkout the verb runs in.
`commit-tree` touches no HEAD, index or working tree, so a detached HEAD, a
dirty tree or a rebase in progress makes no difference (C1). It runs under the
developer's own git configuration (`gitutil.ScrubbedEnv`, as the land stage's
networked git does), never under the isolated configuration `gitutil.Run`
uses, because the author identity lives in the person's global git file; author
and committer are therefore the person's git identity. A checkout whose git has
no `user.name` or `user.email` is refused before anything is written.

**The empty tree.** Every reservation commit's tree is the empty tree, named
for the repository's object format by `git hash-object -t tree /dev/null`
(`4b825dc642cb6eb9a060e54bf8d69288fbee4904` under SHA-1) and written with `-w`
so the object exists before the push. `gitutil.EmptyTree(root)` is the one
place that name is computed; the reader, the writer and the pre-push hook all
compare against it.

**The message.** A subject line, a blank line, then one trailer per line:

```text
abcd reservation: iss-2609020716570699

Abcd-Reservation: iss-2609020716570699
Abcd-Holder: octo
Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55
Abcd-Footprint: internal/core/claim/claim.go
Abcd-Footprint: commands/reserve.md
Abcd-Claimed-At: 2026-10-03T14:00:00Z
Abcd-Expires-At: 2026-10-03T16:00:00Z
Assisted-by: abcd:v0.9.0
```

The intent's names map one to one: The record id is `Abcd-Reservation`, the
holder `Abcd-Holder`, the copy `Abcd-Copy`, the footprint one `Abcd-Footprint`
per path, `claimed_at` is `Abcd-Claimed-At`, `expires_at` is
`Abcd-Expires-At`, and a tombstone adds `released_at` as `Abcd-Released-At`
with `Abcd-Expires-At` equal to it and the subject `abcd reservation released:
<id>`. Trailer keys are hyphenated because git's trailer grammar admits no
underscore. Times are RFC 3339 in UTC, to the second. `Assisted-by:
abcd:<version>` is written by the verb through `assistedby.ComposedValue`,
because `commit-tree` runs no `prepare-commit-msg` hook; it is the
`abcd:<version>` form the attribution gate already accepts (C33).

**The size cap.** A message is at most 4,096 bytes. The writer measures the
message it would write and refuses a larger one before running any git command
(C30). The reader reads the commit object through a bounded read of 8,192 bytes
(`git cat-file commit`), and a commit whose message passes 4,096 bytes, or
whose object passes the bound, is unreadable (C14).

**Public safety.** The writer runs the message through
`scanner.CheckOutbound` and refuses on any finding (no session URL, no tool
footer). Nothing in the message names a host or a home path: The holder is a
username, the copy key is random hex, footprint paths are repository-relative
(`fsutil.ValidRelPath`, which refuses an absolute path, `..` and control
characters), and times are UTC (C28). A footprint path is a trailer value and
nothing else: It is written into the message on standard input and never
becomes a git argument, a refspec or a shell word, so shell metacharacters, a
colon and an asterisk are stored verbatim (C29). Footprints are taken with a
repeatable `--footprint <path>` (`StringArray`, never split on commas).

**The reader treats every value as data.** A tip is parsed by its known
trailer keys from the message's final paragraph; an unknown key is ignored; a
value is shown quoted (`%q` after `termsafe.Sanitize`) wherever it is rendered
and is never placed into a brief or prompt as instruction text. A tip is
**unreadable** when any of these holds: The tree is not the empty tree; the
message or object passes the cap; `Abcd-Reservation` is missing, repeated or
names another record than the branch; `Abcd-Holder` is missing, repeated or
not a GitHub username (`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`);
`Abcd-Copy` is not 32 lowercase hex characters; a time is missing or not RFC
3339; a footprint value fails `fsutil.ValidRelPath`. The reason is carried with
the unreadable state for the listing (C12).

### The branch

The ref is `refs/heads/abcd-reserved/<record-id>`, built only after the
argument passes `recordid.CitedIDRe`, is one of the two reservable families
(`iss-N` or `itd-N`, the records a build or a drain takes), and is put in
canonical form by `recordid.CanonCitedID`, so two spellings of one id never
make two branches (C31). The remote-tracking ref is
`refs/remotes/origin/abcd-reserved/<record-id>`. The prefix is disjoint from the
implement loop's lane prefix `build/`, and the land stage's branch removal
(`update-ref -d refs/heads/<lane.Branch>`) is only ever given a lane branch
(C51).

### The holder and the working copy

**The holder** is the person's GitHub username, resolved in this order and from
nowhere else (`claim.Holder`):

1. `git config user.email`, read under the developer's own configuration, when
   it is a GitHub no-reply address (`<id>+<username>@users.noreply.github.com`
   or `<username>@users.noreply.github.com`). The pattern is the one
   `internal/core/site/contributors.go` already matches, lifted into
   `internal/core/claim` and called by both, so the forge's two forms are
   spelled once (C17).
2. The key `reservation.github_username` in the machine layer of
   `layered.Config`, that is `~/.abcd.noindex/config.json`, read through
   `layered.Load` and taken only from a `Found` whose layer is `Machine`
   (C18). The `reservation` namespace is claimed by this reader, so an unknown
   key under it is refused by the resolver as for every other namespace.

A value for the same key in the repository's `.abcd/config.json` is never
used: With neither user-level source, every write is refused naming both and
saying the repository's own configuration is not read for this (C19). Usernames
compare without regard to case, as the forge treats them, and are written as
resolved. A laptop and a cloud session that resolve to one username are one
person for the cap, the per-person count and the holder check.

**The working-copy key** is 32 lowercase hex characters from `crypto/rand`,
minted the first time a reservation verb runs in a checkout and kept in that
checkout's local tier at `.abcd/.work.local/reservation/copy-key` (mode
`0600`, created with `fsutil.CreateExclusiveIn` so two first runs cannot mint
two keys). It names neither a path nor a host. The checkout is the one
`gitutil.CheckoutRoot` resolves for the verb; the implement loop uses the run's
`RepoRoot`, the checkout the run lives in, so a run's start, renewal, fence and
tidy-up all speak for one copy. The local tier is never created on the way to a
reservation (the precedent of `loop.tierPresent`): A checkout without
`.abcd/.work.local/` is refused, naming `abcd ahoy`, which creates it.

### Reading the remote, and the view

Every reservation write first fetches, in one command when it can:

```text
git fetch --prune --no-tags origin \
  +refs/heads/abcd-reserved/*:refs/remotes/origin/abcd-reserved/* \
  +refs/heads/<default>:refs/remotes/origin/<default>
```

so the tip it judges, the cap and the overlap it counts, and the derived
release it applies are all read from the view fetched just before the push
(scope condition 9). The default branch is the one this clone last knew as
origin's, and that name can be stale (a remote whose `main` was renamed), when
git refuses the whole command for want of the one ref. The reservations are
then fetched alone, and the act goes on without the derived release, saying
the check was skipped (`derived_unchecked`); a failed fetch of the
reservations themselves refuses the act. The default branch is the one `gitutil.DefaultRef`
resolves under `refs/remotes/origin/`. A fetch is a git command that reaches the
remote; it is lifted from the land stage's `netGit` into one runner,
`gitutil.Net`, which both the loop and the claim package call: hooks run, the
developer's own configuration less any injected `GIT_DIR` or `GIT_CONFIG_*`,
`GIT_TERMINAL_PROMPT=0` so git's own credential prompt cannot hang a
session, ssh in batch mode where abcd composes the ssh command (ssh reads the
terminal itself, so a first host-key question would otherwise wait; a
developer's own `GIT_SSH_COMMAND` or `core.sshCommand` is left as it is), a
timeout, and capped output.

After every successful fetch the verb writes the time to
`.abcd/.work.local/reservation/fetched-at`. **The view's age** is now minus
that time; with no stamp the view is "nothing fetched here", whatever
remote-tracking refs a plain `git fetch` may have left, so a render never says
an item is free on a view it cannot date (C50). A view made by a plain `git
fetch` under the default refspec carries the reservations too (scope condition
11); it is shown, and its age is reported as unknown.

`claim.View(root, now)` reads the view from disk only (`git for-each-ref`,
`git cat-file`, `git ls-tree`; no fetch, no push) and returns every reservation
with its holder, copy, footprint, effective expiry, state, and the view's age.
It is the "as of last fetch" reader the resolve-time refusal
(itd-2609091416304128) is to reuse.

### One liveness judgement

`internal/core/lease` is a leaf package holding `Default` (two hours), `Min`
(one minute) and `Max` (24 hours, the shipped `MaxLease`), and one function:

```go
// Judge says whether a claim holds at now. A readable claim's effective expiry
// is min(ExpiresAt, ClaimedAt + Max); an unreadable one is held until
// WrittenAt + grace.
func Judge(s Stamp, grace time.Duration, now time.Time) Verdict
```

`Verdict` carries `EffectiveExpiry`, `Live` and `Unreadable`. The in-run claim
calls it with its one-minute `UnreadableClaimGrace` and the claim file's
modification time; the reservation calls it with the two-hour default lease and
the tip's committer date. `implement.DefaultLease`, `MinLease` and `MaxLease`
become the lease package's values under their existing names, so every
`implement claim` test passes unchanged (C32). A tip whose `Abcd-Expires-At`
is thirty days after its `Abcd-Claimed-At` is live until claimed-at plus 24
hours (C10); a requested lease outside one minute to 24 hours is refused before
any git command runs (C11).

**The takeover margin** is five minutes, the reservation's own constant
(`claim.TakeoverMargin`): Another person may take over a reservation only once
its effective expiry (or, for an unreadable tip, the end of its hold) passed at
least five minutes ago by the taker's clock (C8, C9, C13).

**The derived release.** Before judging liveness, the reader asks whether the
record is terminal on `refs/remotes/origin/<default>` as last fetched: an issue
in `.abcd/work/issues/resolved/` or `wontfix/`, an intent in `shipped/` or
`superseded/`. It reads the tree through the peers package's tree reader
(`scanTree`, exported as `peers.FoldersAt(root, ref)`), so one reader decides
which folder a record is in. A terminal record's reservation counts as
released, whatever its expiry (C15). With no default branch on `origin`, nothing
is derived.

The states a listing shows are `live`, `expired`, `released` (a tombstone, or
derived from the default branch, said which), and `unreadable` (with its reason
and the end of its hold).

### The push semantics

Every act is fetch, judge, write one commit, then one plain push:

```text
git push --porcelain origin <commit>:refs/heads/abcd-reserved/<id>
```

with the repository's hooks running. No `--force`, no `+` refspec, no
`--force-with-lease`, no `--no-verify`, no deletion: The hazard registry
refuses the first four and a hosted cloud session's proxy refuses deletions,
while a plain push is already the compare-and-swap.

| Act | When the tip is | The commit pushed |
|---|---|---|
| Reserve | absent | parentless |
| Reserve | a tombstone, or released by the default branch | child of the tip (C7) |
| Renew | this person and this copy, live | child with a later `Abcd-Expires-At` (C4) |
| Take over from one's own other copy | this person, another copy, live, with `--take-over` | child naming this copy; the output names the copy replaced (C52) |
| Take over | another person's, or unreadable, past its expiry or hold by at least the margin | child; the output names the holder or the unreadable commit and the expiry replaced (C8, C13) |
| Release | this person and this copy, not already a tombstone | tombstone child (C6) |

Refused, nothing pushed:

- The tip is another person's and live, or past its expiry by less than the
  margin: the back-off exit, naming the holder and the effective expiry (C2,
  C9).
- The tip is this person's, from another copy, live, and `--take-over` is
  absent: the back-off exit, with the confirmed message (C20).
- A release whose tip is another person's, or another copy's (C5).
- `--take-over` never overrides another person's live reservation: It moves
  only one's own.

**A lost race.** When the push is rejected, the verb fetches again and judges
the new tip. If the branch now holds a reservation it cannot take (the parentless
create met an existing branch, or another child landed first), it refuses with
the back-off exit naming that holder, and says the reservation was taken while
this push was on its way (C3). This covers both ways the race shows: git's
own client refusing the push as a non-fast-forward once the remote's
advertisement shows the moved tip, and the remote's own compare-and-swap
refusing the update when the other push lands after the advertisement, while
this one is being received. A pre-push hook is never asked in a lost race:
git rejects the ref before it runs the hook. Otherwise the push failure is
reported as itself.

**Failure writes nothing locally.** A missing `origin`, an `origin` path that
does not exist, no write permission, offline, or a remote or proxy that rejects
the push: Each refuses naming its cause, quotes git's own rejection (with the
home directory redacted by `fsutil.RedactHome`), and says that no reservation was
made (C21, C22, C23). The commit `commit-tree` wrote is an unreferenced object
that git's own garbage collection removes; no ref, file or log line records a
reservation that did not reach the remote.

**Only `origin`.** A checkout whose remotes do not include one named `origin`
is refused, naming the remotes found, or saying there are none (C21).

### The cap, the footprint and the overlap (spec 2 for the cap and overlap)

- **The footprint is required.** A reservation with no `--footprint` is
  refused (C27). This is spec 1.
- **The per-person cap is ten.** A reservation that would take its holder past
  ten live reservations (counted across every record in the fetched view, by
  username, released and expired ones excluded) is refused, naming the cap and
  the holder's live reservations (C47). A renewal or a take-over of one's own
  other copy adds none and is never refused by the cap.
- **Overlap refuses.** A new reservation whose footprint shares a path with any
  other live reservation (another record, whoever holds it) is refused with the
  back-off exit, naming that reservation, its holder and the shared path
  (C48). Two paths are shared when they are equal or one is a parent directory
  of the other, segment by segment, because a spec's footprint names package
  directories while an issue's names a file. A renewal re-checks against the
  view, so a footprint is never widened past another live reservation.

### The verbs

**`abcd reserve`** is a new top-level verb, the front door onto
`internal/core/claim` for the work no build or drain starts. It is an agent's
verb (the product thinker's ruling of 2026-10-12, under "Settled here"): it is
listed in the agents-and-hosts block `abcd --help --agent` renders
(`internal/surface/cli/helpgroups.go`), naming `commands/reserve.md` as the
page to read next, its sentence is in `sentences.go`, and its plugin page
carries `user-invocable: false`, which keeps it out of a person's `/abcd:`
menu. It stays runnable from the CLI. A person sees reservations on the board
and a record's page, the spec-2 renders (`abcd peers`, which lists them, is
an agent's verb as well), not through the verb:

```text
abcd reserve <record-id> --footprint <path> [--footprint <path>…] [--take-over] [--lease <duration>] [--json]
abcd reserve <record-id> --release [--json]
```

It is wired on both surfaces in one change, per AGENTS.md "Wired or it isn't
done": `newReserveCommand` in `internal/surface/cli/reserve.go`, registered in
`cli.go`, and the plugin page `commands/reserve.md` naming the push as the
verb's documented meaning (invariant 7). The surface-parity test reads both. A
bare `abcd reserve` refuses as a usage error and points at `abcd peers`, which
lists. `--release` with `--footprint`, `--take-over` or `--lease` is a usage
error. `--json` emits the act (`reserved`, `renewed`, `taken_over`,
`released`), the record, the holder, the copy (`this` or `another`, never the
key of another copy), the footprint, the effective expiry, and what it
replaced.

**`abcd build <itd-N> [--release] [--take-over]`** (spec 2). A new run, and a
resumed one, reserves the intent at `loop.start`, after the pre-start checks
and before the lane opens, with the footprint the spec's `## Footprint`
`packages:` names (`spec.ReadFootprint`). A spec with no footprint refuses the
start at the reservation, naming the spec and the missing section, and starts
nothing (C34, C35). `--release` pushes the tombstone and changes nothing in the
run's state; a later step meets it at the fence (C36). `--take-over` passes
through to the reservation. `build next` reaches the same `start`, so the
picked intent is reserved the same way.

**`abcd drain`** (spec 2) reserves each issue as its lane starts, with the
issue's `found_at` path as the footprint (C37). An issue whose `found_at` is
empty or not a repository-relative path is refused at the reservation; the
drain hands it back with that reason and moves to the next issue.

**`abcd peers`** (spec 1) gains its first network act: It fetches the
reservation branches with pruning (and the default branch, for the derived
release) and lists every reservation with its record, holder, footprint,
effective expiry and state, and each person's count of live reservations, below
the sibling-worktree rows it already prints (C12, C16). `--json` gains
`reservations`, `live_by_holder` and `view` (`fetched_at`, `age_seconds`). When
the fetch fails, the listing still renders the view as last fetched with its
age, names the failed fetch on stderr and in `fetch_error`, and exits 1; the
worktree rows are unchanged. Its sentence and `commands/peers.md` say it
fetches.

### `implement step`: renew, fence and tombstone (spec 2)

The run's reservation is the run's key (`State.Key`, an intent or an issue),
held for the run's checkout's copy.

- **Renewal.** `advance`, before it performs a stage, reads the tip from the
  view and, when the tip is this person's and this copy's and half its lease has
  passed since its `Abcd-Claimed-At`, fetches and pushes a renewal with the
  default lease, logging a `reservation_renewed` event in the run log (C40).
  With less than half passed, it pushes nothing (C41). A renewal that fails
  outside the land stage is named in the step's note and logged, and the stage
  still runs: the fence is where a lost reservation stops the run.
- **The land fence.** At the top of `landArm`, before the ruleset decides
  between arming the merge and leaving the pull request for a person, the stage
  fetches the reservation and refuses unless the tip is this person's and this
  copy's and live: it names the other person, or says the reservation moved to
  this person's other copy, or says it was released, and arms nothing (C42,
  C53). When it holds, the stage renews it with a lease of
  `max(lease.Default, timeout + 15 minutes)`, clamped to `lease.Max`, where
  `timeout` is `check_response_timeout_minutes` read from the ruleset mirror at
  the lane's base (`.abcd/work/rulesets/`, through the same tree read
  `mergeRule` makes; 45 here). With no mirror, the lease is the default two
  hours (C43, C44). While `landMerged` waits, each step renews at half-lease by
  the rule above.
- **The tidy-up.** In `landMerged`, once the ancestor check proves the lane
  landed, the reservation has already stopped counting (the derived release).
  The step then pushes a tombstone when the tip is still this person's and this
  copy's (C45), and otherwise leaves the branch and names the holder in its note
  (C46). A failed tombstone push is named in the note and never undoes the
  landing.
- **The lane branch** removal is unchanged and touches only `build/<run>`
  (C51).

### Renders read only the disk (spec 2)

`abcd <record-id>` (`record.Describe`), the board (`loop.StatusPeers`, read by
`internal/surface/cli/board_status.go`), `build <itd-N>`'s `peers` check and
`build next`'s exclusions read the view through `claim.View`, judged by
`lease.Judge`, with no fetch and no push (C49):

- `Description` gains a `reservation` member: holder, `this copy` or `another
  copy` when it is the reader's own, effective expiry, state and the view's
  age. A view that was never fetched renders "reservation: unknown, nothing has
  been fetched here; `abcd peers` fetches" rather than "none" (C50).
- The board's held-by text for an intent names the reservation and the view's
  age; the site's status block passes no peer reader and so never shows one.
- `peersCheck` gains the reservation as a holder: "reserved by octo until
  15:00Z, as of the last fetch 12 minutes ago". `build next` lists it under
  `excluded` with that reason (C38) and `build <itd-N>` refuses at the `peers`
  check with it (C39). The run's own reservation (this person, this copy) is
  never a holder.

### The pre-push hook (spec 1)

`.githooks/pre-push` gains one exemption, placed after the non-fast-forward
guard and before the receipt check: An update to `refs/heads/abcd-reserved/*`
passes without a preflight receipt only when every commit in the pushed range
has the empty tree. The range is `<remote_sha>..<local_sha>` on an update and
every commit reachable from `<local_sha>` on a create, so a branch of ordinary
history pushed under the prefix walks into a non-empty tree and is gated as
today (C25). The empty tree is computed with `git hash-object -t tree
/dev/null`. The non-fast-forward refusal stays and covers reservation branches
too (C26). No reservation push uses `--no-verify` (C24). The hook's header
comment gains the exemption and its reason.

`internal/core/guard/defaults/guard.json` gains a reservation push,
`git push --porcelain origin 9f2c41d07be35a6c80e1f4b2a7d39c559f2c41d0:refs/heads/abcd-reserved/iss-2609020716570699`,
as a `known_good` of `git-push-force-refspec` and `git-push-force`, so the
registry is shown to pass the one push the verb makes.

### Every refusal

Each line is the message after the verb's prefix (`abcd reserve:`, `abcd
build:`, `abcd drain:`, `abcd implement step:`). Exit 3 is the back-off exit,
as `implement claim` uses it; exit 2 is every other refusal and every failure,
and each ends by saying nothing was pushed or no reservation was made.

| Case | Exit | Message |
|---|---|---|
| Not a reservable id | 2 | `"<arg>" is not an issue or intent id (iss-N or itd-N); nothing was pushed` |
| No footprint | 2 | `a reservation names the files the work will change: pass --footprint <path> once per path; nothing was pushed` |
| Bad footprint path | 2 | `footprint "<path>" is not a repository-relative path; nothing was pushed` |
| Lease out of bounds | 2 | `lease <d> is outside 1m0s to 24h0m0s; nothing was pushed` |
| Message over the cap | 2 | `the reservation message would be <n> bytes, over the 4096-byte cap: name fewer or shorter footprint paths; nothing was pushed` |
| Outbound finding | 2 | `the reservation message carries <class>, which public text may not carry; nothing was pushed` |
| No username | 2 | `cannot tell which GitHub user you are: user.email is not a GitHub no-reply address, and ~/.abcd.noindex/config.json sets no reservation.github_username; set one of them (the repository's own configuration is never read for this); nothing was pushed` |
| No git identity | 2 | `git has no user.name or user.email to author the reservation; set them, then retry; nothing was pushed` |
| No local tier | 2 | `.abcd/.work.local/ is not a directory in this checkout, so there is nowhere to keep this copy's key; run abcd ahoy to set it up; nothing was pushed` |
| No `origin` | 2 | `reservations live on the remote named origin, and this checkout's remotes are: <names>; nothing was pushed` (or `this checkout has no remote`) |
| Fetch failed | 2 | `could not read the reservations from origin: <git's reason>; no reservation was made` |
| Push rejected | 2 | `origin refused the reservation push: <git's rejection>; no reservation was made` |
| Held by another person | 3 | `<id> is reserved by <holder> until <expiry>; back off and take other work` |
| Lapsed under the margin | 3 | `<id> is reserved by <holder>, lapsed at <expiry>, under five minutes ago by this computer's clock; retry after <expiry + 5m>` |
| Unreadable within its hold | 3 | `the reservation on <id> is unreadable (<reason>); it is held until <commit date + 2h> and can be taken over after <+5m>` |
| Lost the race | 3 | `<id> was reserved by <holder> while this push was on its way, until <expiry>; back off and take other work` |
| Another copy of yours | 3 | `you are already building <id> on another computer, until <expiry>; pass --take-over to continue here` |
| Release of another's | 2 | `<id> is reserved by <holder>, not by you; nothing was pushed` |
| Release from another copy | 2 | `<id> is held by your copy on another computer; release it there, or take it over here first; nothing was pushed` |
| Footprint overlap | 3 | `"<path>" is in the footprint of <other-id>, reserved by <holder> until <expiry>; take other work; nothing was pushed` |
| Over the cap | 2 | `you hold 10 live reservations, the most one person may hold: <ids>; release one (abcd reserve <id> --release) first; nothing was pushed` |
| Build with no footprint | 2 | `<spc-N> has no footprint under ## Footprint to reserve <itd-N> with; fill its packages line, then run abcd build again; nothing was started` |
| Drain issue with no `found_at` | handed back | `<iss-N> has no found_at path to reserve it with; handed back` |
| Fence: another person | contention | `the reservation on <id> is now <holder>'s, until <expiry>, so this lane does not arm its merge` |
| Fence: another copy | contention | `the reservation on <id> moved to your copy on another computer, so this lane does not arm its merge here` |
| Fence: released | contention | `the reservation on <id> was released at <time>, so this lane does not arm its merge; reserve it again with abcd build <id>` |
| Fence: fetch failed | contention | `could not read the reservation on <id> from origin: <git's reason>, so the merge is not armed; run abcd implement step again once origin answers` |

A release whose tip is already this copy's tombstone exits 0, saying it was
already released, and pushes nothing.

## How each acceptance criterion is met

Criteria are numbered C1 to C53 in the intent's order: C1 to C33 under spec 1,
C34 to C53 under spec 2. "Step" names this spec's `## Steps`; a test name is
the test that proves the criterion, in the package the step lands in.

| C | Step | Test |
|---|---|---|
| C1 | 3 | `claim.TestAReservationIsAParentlessEmptyTreeCommitAndLeavesTheCheckoutAlone` |
| C2 | 3 | `claim.TestAnotherPersonsLiveReservationIsRefusedWithTheBackOffExit` |
| C3 | 3 | `claim.TestAStaleViewLosesTheRaceAndNamesTheWinner` (the advertised tip, and the push landed while this one is received) |
| C4 | 3 | `claim.TestAFreshAgentInTheSameCopyRenews` |
| C5 | 3 | `claim.TestReleasingAnotherPersonsReservationPushesNothing` |
| C6 | 3 | `claim.TestReleasePushesATombstoneAndDeletesNothing` |
| C7 | 3 | `claim.TestATombstoneIsReservedAsItsChild` |
| C8 | 3 | `claim.TestAReservationLapsedPastTheMarginIsTakenOverNamingWhatItReplaced` |
| C9 | 3 | `claim.TestAReservationLapsedUnderTheMarginIsRefused` |
| C10 | 1 | `lease.TestTheEffectiveExpiryIsClampedToADay` |
| C11 | 2 | `claim.TestALeaseOverADayIsRefusedBeforeAnyGit` |
| C12 | 2, 6 | `claim.TestATipIsReadAsDataAndAMalformedOneIsUnreadable`; `cli.TestPeersQuotesAHostileReservation` |
| C13 | 3 | `claim.TestAnUnreadableTipPastItsHoldIsTakenOverNamingTheCommit` |
| C14 | 2 | `claim.TestAMessageOverTheCapReadsAsUnreadable` |
| C15 | 3 | `claim.TestATerminalRecordOnTheDefaultBranchReleasesItsReservation` |
| C16 | 6 | `cli.TestPeersFetchesAndListsEveryReservationWithCounts` |
| C17 | 2 | `claim.TestTheHolderIsReadFromANoReplyAddress` |
| C18 | 2 | `claim.TestTheHolderIsReadFromTheMachineConfig` |
| C19 | 2 | `claim.TestARepositoryConfiguredUsernameIsNeverUsed` |
| C20 | 3 | `claim.TestAnotherCopyOfTheSamePersonIsRefusedWithoutTakeOver` |
| C21 | 3 | `claim.TestARemoteNotNamedOriginIsRefusedNamingTheRemotesFound` |
| C22 | 3 | `claim.TestAMissingOriginFailsAndMakesNoReservation` |
| C23 | 3 | `claim.TestARejectingRemoteIsQuotedAndMakesNoReservation` |
| C24 | 4 | `cli.TestPrePushPassesEveryReservationActWithoutAReceipt` |
| C25 | 4 | `cli.TestPrePushGatesANonEmptyTreeUnderTheReservationPrefix` |
| C26 | 4 | `cli.TestPrePushRefusesANonFastForwardReservationPush` |
| C27 | 5 | `cli.TestReserveWithoutAFootprintIsRefused` |
| C28 | 2, 5 | `claim.TestAReservationNamesNoHostNoHomeAndPassesTheOutboundPolicy`; `cli.TestReserveRendersCarryNoHostOrHomePath` |
| C29 | 2 | `claim.TestAHostileFootprintIsStoredVerbatimAndNeverAnArgument` |
| C30 | 2 | `claim.TestAMessageOverTheCapIsNeverWritten` |
| C31 | 2 | `claim.TestAnInvalidIDIsRefusedBeforeAnyRefName` |
| C32 | 1 | `implement.TestClaimJudgesThroughTheSharedLease` and the existing `implement` claim tests, unchanged |
| C33 | 4 | `claim.TestReservationCommitsPassTheAttributionGate` |
| C34 | 13 | `loop.TestBuildReservesTheIntentWithItsSpecFootprint` |
| C35 | 13 | `loop.TestBuildWithoutAFootprintIsRefusedAtTheReservation` |
| C36 | 13 | `cli.TestBuildReleasePushesATombstone` |
| C37 | 14 | `loop.TestDrainReservesEachIssueWithItsFoundAt` |
| C38 | 12 | `loop.TestBuildNextExcludesAnotherPersonsReservationWithTheViewsAge` |
| C39 | 12 | `loop.TestBuildRefusesAnotherPersonsReservationAtThePeersCheck` |
| C40 | 8 | `loop.TestAStepRenewsAtHalfLease` |
| C41 | 8 | `loop.TestAStepUnderHalfLeasePushesNothing` |
| C42 | 8 | `loop.TestTheFenceRefusesAReservationNowAnotherPersons` |
| C43 | 8 | `loop.TestTheFenceRenewsPastTheQueueTimeout` |
| C44 | 8 | `loop.TestTheFenceRenewsWithTheDefaultLeaseWithoutAMirror` |
| C45 | 9 | `loop.TestALandedLaneTombstonesItsOwnReservation` |
| C46 | 9 | `loop.TestALandedLaneLeavesAnotherPersonsReservation` |
| C47 | 10 | `claim.TestTheEleventhReservationIsRefusedNamingTheCap` |
| C48 | 10 | `claim.TestAnOverlappingFootprintIsRefusedNamingTheSharedPath` |
| C49 | 11, 12 | `cli.TestReservationRendersTouchNoNetwork` (the record and the board in step 11; `build <itd-N>` and `build next` added in step 12) |
| C50 | 11 | `record.TestANeverFetchedViewSaysSoRatherThanFree` |
| C51 | 9 | `loop.TestLaneBranchRemovalLeavesTheReservationBranch` |
| C52 | 3, 5 | `claim.TestTakeOverMovesTheReservationToThisCopy`; `cli.TestReserveTakeOverNamesTheCopyReplaced` |
| C53 | 8 | `loop.TestTheFenceRefusesAfterTheOtherCopyTookOver` |

**The test harness.** Every criterion is proved with a local bare repository
standing in for `origin` (`git init --bare`, as `land_test.go` stands one up),
further clones standing in for the second person, the second computer and the
cloud session, each with its own `HOME`, `user.email` and local tier, and an
injected clock (`claim.Options.Now`, as the run's `Options.now`). A rejecting
remote is a bare repository whose `pre-receive` hook refuses the prefix. The
zero-network test (C49) follows `TestModeAndStatuslineTouchNoNetwork`: it
counts requests through `http.DefaultTransport` and, after the clone has
fetched the reservations, moves the bare repository away, so any fetch or push
a render made would fail the test.

## Settled here, not by the intent

These are the facilitator's design calls; none changes a criterion.

- Trailer keys are prefixed `Abcd-` and hyphenated, because git's trailer
  grammar admits no underscore and the prefix keeps them apart from every other
  trailer gate.
- Only `iss-N` and `itd-N` are reservable: They are the records a build or a
  drain takes. The record need not exist in the reserving checkout.
- Two footprint paths are shared when equal or when one is a parent directory
  of the other, segment by segment.
- The copy key lives in the local tier and the tier is never created by a
  reservation verb; a fresh clone runs `abcd ahoy` first.
- The view's age comes from a stamp abcd writes after its own fetches; a view
  made only by a plain `git fetch` is shown with its age unknown.
- A renewal that fails outside the land stage is reported and does not stop
  the stage; the fence stops the run.
- `abcd peers` exits 1 when its fetch fails, after rendering the last view.
- `build --release` changes nothing in the run's state.
- **`abcd reserve` is an agent's verb** (the product thinker's ruling,
  2026-10-12, in their words: "I suggest /abcd:reserve becomes an agent-only
  verb"). It is listed in the agents-and-hosts block, not in a person's
  group, and `commands/reserve.md` carries `user-invocable: false`; the page is
  still the plugin surface, so the verb is wired on both surfaces. It stays
  runnable from the CLI, and a person sees reservations through the record
  and board renders; `abcd peers` is an agent's verb too.
- **What steps 1 to 3 settled** (lane 1's build, recorded with step 7):
  - The machine layer is `~/.abcd.noindex/config.json` (the intent has it
    right; this spec said `~/.abcd/config.json`). Every message naming it
    takes the path from `layered.Config.MachineOrigin()`.
  - A renewal resets `Abcd-Claimed-At` to now, so spec 2's half-lease rule
    and the 24-hour clamp count from the renewal, not from the first
    reservation.
  - A tombstone's `Abcd-Claimed-At`, `Abcd-Expires-At` and `Abcd-Released-At`
    are all the release's time, and it keeps the tip's footprint.
  - The same person's other copy, past its effective expiry plus the
    five-minute margin, is taken over without `--take-over`, as another
    person's lapsed reservation would be.
  - Two release refusals the table had no row for: with no tip, "`<id>` holds
    no reservation on origin to release; nothing was pushed"; with an
    unreadable tip, "the reservation on `<id>` is unreadable (`<reason>`), so
    nobody holds it to release; nothing was pushed". A release over any
    tombstone, whoever wrote it, exits 0 as `already_released`.
  - The derived release applies to an unreadable tip too: a record finished
    on the default branch releases its reservation whatever its tip says or
    fails to say.
  - C30's "before any git": the size check runs after `git config user.name`
    and `user.email` are read, because the holder's length is part of the
    message; it still runs before any write, fetch or push, and the test
    checks that only config reads came first.
  - `commit-tree` runs with `gitutil.ExecPins()`, so no signing program or
    hook starts, with author and committer set from the person's `user.name`
    and `user.email` and dated by the injected clock. `gitutil.Net` returns
    stdout on a failure too, where a porcelain push reports its rejections.
  - The in-run claim reads a claim file with no `claimed_at` as unreadable,
    since the 24-hour clamp needs a starting point.
  - `peers` names the ledger directory through `recordid` rather than
    `capture`, which breaks the import cycle `site → claim → peers → capture →
    site`.
- **What the review of steps 1 to 3 and steps 4 to 7 settled** (2026-10-12):
  - The size cap measures a reservation's tombstone too (about 48 bytes
    longer), so a reservation the writer accepts can always be released.
  - An unreadable tip too big to read whole takes its committer date from git.
    One whose date cannot be read at all (an object that is not a commit, or a
    committer line that does not parse) is held from when this copy first
    fetched it, kept in the local tier at
    `.abcd/.work.local/reservation/undated/<commit>` so a later fetch does not
    restart the hold, and the refusal says the hold is counted from then.
    `claim.View`, being a reader, reads that time and records none.
  - The networked runner's ssh runs in batch mode where abcd composes the ssh
    command, so a host-key question is a failure, not a ten-minute wait.
  - The attribution gate read a commit with no parent as a silent exit 1 (its
    last identity read met end of input under `set -e`); a reservation's
    create is such a commit, so the gate was fixed and its case corpus holds a
    parentless commit both ways.
  - The brief chapter is `37-reserve.md`, because `36-dashboard.md` took 36.
  - `abcd peers` in a checkout with no remote named `origin` lists no
    reservations and exits 0, as before: there is nothing to fetch, which is
    not a failed fetch. In a checkout without its local tier, the fetch is
    made but not stamped, so the view's age is unknown. Its sentence now says
    it writes the fetched reservations.
  - The pre-push hook's create range is every commit reachable from the
    pushed tip, as written.
- **What the security review of steps 4 to 7 settled** (2026-10-12):
  - The hook reads git's null object id by what it is made of, all zeros of
    any length, so a SHA-256 repository's sixty-four zeros are read as SHA-1's
    forty are: a reservation create there passes without a receipt, and a
    delete there, of a reservation or of the protected branch, is read as a
    delete. The protected-branch delete check had the same forty-zero
    constant, so a SHA-256 delete of it was refused under the wrong reason.
  - The hook refuses a deletion of any ref under the reservation prefix,
    naming `abcd reserve <id> --release`, so "no deletion" holds for a push
    made through the hooks. A push with `--no-verify`, or from a checkout
    without them, is held by the server: the ruleset "reservations
    protection" on `refs/heads/abcd-reserved/**` refuses deletion and
    non-fast-forward with no bypass actors. It is applied, ruled by the
    product thinker on 2026-10-12, and mirrored in
    `.abcd/work/rulesets/reservations-protection.json`. A later sweep that
    deletes reservation branches (itd-118, out of scope here) needs that
    ruleset amended first. No hazard-registry entry was added: the registry's operand
    prefixes are all-of, so one entry per spelling (two colon refspecs, two
    `--delete` operands) would be needed, each taught as its own lesson, and a
    deletion by `--prune` or `--mirror` cannot be named at all; the hook sees
    every deletion however it was spelled.
  - The hook's exemption is a declared table of receipt-less prefixes, one
    row per prefix: the prefix, the shape check a push under it must pass,
    the record that rules it, and the act that ends a ref there, named when
    a delete is refused. The reservation prefix is its only row, ruled by
    adr-2610031352212697, with the empty-tree walk plus the fast-forward the
    non-fast-forward guard already demands as its shape check. A new row
    needs its own ruling record and its own shape check, run through its own
    arm in the hook's `shape_check`, so no command runs under a name read
    from the table; a row without a shape check, or naming one with no arm,
    exempts nothing, and a prefix the table does not name needs a receipt.
    The next row expected is cloud sessions' work-in-progress branches,
    ruled by adr-2610120937435856, which spc-2610120941441282 adds. <!-- record-lint: forward-looking -->
- **What step 10 settled** (the cap and the overlap, 2026-10-12):
  - Both are judged after the push table has chosen the act and before the
    commit is written, on the view the act just fetched, read by the same
    reader as `claim.View`; the derived release is left out when the act's
    fetch could not read the default branch, as the act's own judgement
    leaves it out. The cap is checked first: taking other work does not lift
    it.
  - The cap counts the holder's live reservations on every other record. An
    act over a tip this person already holds, readable, unreleased and live
    (a renewal, or a take-over of their own other copy), adds none and is
    never refused by it. Reviving one's own lapsed tip adds one, since the
    cap does not count it while it is lapsed: ten live and a revived one
    would otherwise make eleven (the review of 2026-10-14; this line first
    said "lapsed or not"). A holder past ten, which only a race
    between two of their copies can make, is told "past the 10 one person
    may hold" rather than the table's "the most one person may hold".
  - The overlap counts every other live reservation but this copy's own on
    another record: the holder's from another copy is counted. (This line
    first counted the holder's own from this copy too; the review of
    2026-10-14 exempted it, since a drain then passed over its own next
    issue when two issues named one `found_at`.) The path named is the deeper of the two shared
    paths, the one in both footprints: a request for `commands` over a
    reservation of `commands/reserve.md` names `commands/reserve.md`.
  - An unreadable reservation is counted by neither: it has no holder and no
    footprint that can be read. It still holds its own record.
- **What step 11 settled** (the record and board renders, 2026-10-12):
  - The board's reservation text is shown and never judged: a row's
    `reserved` member is added from disk after the block is read, and it does
    not move the head. The head follows the build's peers check
    (`loop.StatusPeers`), which gains the reservation in step 12, so the head
    and `build next`'s pick stay one between the two steps (ruling CC1). The
    board therefore reads `claim.View` in `board_status.go`, and step 11
    changes nothing in `internal/core/implement/loop`.
  - The board marks every row, Now, Next and Later, a run's issue row
    included, with the reservation another copy holds on its record: a live
    one, or an unreadable one within its hold. This copy's own is never
    marked. The board does not resolve the reader's username, so the
    reader's other copy is marked as any holder is, by name.
  - `abcd <record-id>` carries the line for an issue or an intent only, and
    only in a checkout with a remote named `origin`: with no such remote
    there are no reservations to show, and no line rather than `unknown`.
    Its `copy` is `this copy`, or `another copy` when the holder is the
    username the reader resolves to by the reservation verb's own rule; a
    view that cannot be read renders `unknown` with the reason.
  - The view's age is worded in whole units: under a minute, minutes below
    two hours, hours below two days, then days.
- **Reservations are a cooperative model, not an access control** (the
  reviewers' reading of 2026-10-14, stated here plainly): they coordinate
  people who share push access in good faith. Any writer with push access
  can append a commit to another holder's reservation branch, and the holder
  is read from a self-declared trailer, set from the writer's `user.email` or
  machine configuration. The forge's push permissions are the only boundary;
  the pre-push hook's checks and the reservations ruleset keep honest tools
  to the protocol and refuse deletion and force, but prove nobody's identity.
- **What the review of steps 8 to 11 and 14 settled** (2026-10-14):
  - A footprint path is a value a writer chose, so the overlap refusal names
    the shared path quoted (`%q` after `termsafe.Sanitize`, as every other
    remote value is echoed), and the drain sanitises the reason it passes an
    issue over with, since the agent driving it reads it. A footprint must be
    printing runes only, with no C1 control, bidi mark or zero-width rune:
    the writer refuses another, and the reader reads a tip carrying one as
    unreadable, which neither the cap nor the overlap counts.
  - A renew-only act (the loop's renewal and its fence) over a live tip is
    not checked for overlap (one over its own lapsed tip is, since the
    lapsed reservation held nothing and another may have reserved its paths
    since; the final review of 2026-10-14): it keeps the tip's own footprint and cannot widen it, and the
    overlap is not compared-and-swapped, so two people racing can each pass
    it with overlapping footprints; were renewals checked, both would then
    fail, and both land fences refuse, until one lapsed. A reserve naming its
    footprint, a renewal by the verb among them, is still checked, so a
    footprint is never widened onto another.
  - A lane parked at land waiting on a person's merge (the final review,
    2026-10-14, a blocking finding): nothing stepped a drain's parked run,
    so its reservation lapsed about two hours after the fence and another
    account's drain could take the issue while it was still open on main.
    Every drain move now renews, at half its lease, the reservation of each
    issue run parked at land (`finishMerged`, logged in the run's record),
    and the fence renews for `lease.Max` under a ruleset mirror (below). Past the fence a reservation holds
    only while something renews it: a step of the run, whatever its stage, an
    implementer's long stage included, or a drain move over its parked lane.
    A run neither stepped nor drained over for longer than its lease loses
    it; that limit stands and is stated on the implement page.
  - The fence's lease is `lease.Max` wherever a ruleset mirror exists at the
    lane's base, and the default lease with none (decision 29, C44). Under a
    mirror every landing the loop makes waits on a person: it arms a merge
    only behind a required approval, and otherwise leaves the pull request
    open for a person to merge, and either can wait overnight (the last
    review, 2026-10-14). A day is past any `check_response_timeout_minutes`
    and its margin, so C43 holds; the queue's timeout no longer decides the
    lease, and the mirror's timeout is not read. The no-mirror case keeps
    decision 29's default rather than the day the reviewers asked for every
    unarmed pull request, since decision 29 is the intent's and changing it
    is its owner's call; such a run's pull request holds by the renewals.
  - A renewal keeps the tip's own term, `min(max(lease.Default, effective
    expiry − Abcd-Claimed-At), lease.Max)`, never the default alone: a
    renewal at half a day-long fence lease would otherwise cut the hold to
    two hours, and a pull request waiting overnight would be lost two hours
    after this account's drain stopped (the last review, 2026-10-14).
  - `implement receipt` renews as a step does, so an implementer working past
    half the lease with no step between keeps the reservation. A step or a
    receipt whose view shows the reservation held by another person or copy
    no longer passes over it in silence: its answer names the holder, and the
    run record logs `reservation_lost` once until something else about the
    reservation is logged (the last review, 2026-10-14).
  - Clock skew is bounded by the take-over margin and documented, not
    corrected. The expiry is the holder's clock judged by the reader's; the
    committer date is written by the same clock, so it bounds nothing. A
    reader-side bound was weighed: holding a tip at least a lease (or
    `lease.Min`) from when this copy first saw its commit. It cannot tell a
    holder's slow clock from a late fetch, so the lease-length form would
    revive, for a copy fetching for the first time, every reservation that
    lapsed long ago, and the `lease.Min` form protects a skewed holder for six
    minutes at the cost of a stamp per tip commit in the local tier, every
    renewal a new one. Neither was built. The bound, stated on the reserve
    page and its chapter: a holder whose clock runs behind by more than half
    its lease plus the margin (65 minutes for the default) can be taken over
    between its own renewals, and a reservation never renewed is open to a
    take-over once its holder's clock is behind by more than its lease; a
    clock ahead is clamped to a day from the reader's now.
  - The cap and the overlap are best-effort under a race: each is judged on
    the view fetched just before the push, and each reservation is its own
    ref, so two pushes for two records cannot see each other. A collision on
    one record is atomic: the plain push is the compare-and-swap.
- **What step 8 settled** (renewal and the land fence, 2026-10-14):
  - The run records its reservation in its state, `reserved_at` (the state's
    schema version 11; version 10 is read as a run holding none). A run whose
    state records none, one started before this build or in a checkout with
    no remote named `origin`, is neither renewed, fenced nor tombstoned. Until
    step 13 makes `abcd build` reserve at start, only a drain's run (step 14)
    holds one.
  - The renewal and the fence are a renew-only act of `claim.Reserve`
    (`Request.Renew`): it renews this person's reservation from this copy and
    refuses every other tip with a `*claim.NotHeld` naming why (absent,
    unreadable, released, another person's, another copy's), so a run's
    renewal never reserves a released record again, which keeps `build
    --release` a release (C36), and never takes one over. It keeps the tip's
    footprint and is never counted by the cap. It was first also checked for
    overlap; the review of 2026-10-14 dropped that (below): it cannot widen
    the footprint, and an overlap a race let in would stop both renewals and
    both fences until one lapsed.
  - The step's renewal renews one's own lapsed reservation too, since half its
    lease has passed. The fence requires a live one (`Request.Live`) and
    refuses a lapsed one; a step meets that only when its own renewal, which
    runs first in the same step, failed.
  - Half the lease is counted on the effective lease, `Abcd-Claimed-At` plus
    half the time from it to the effective expiry.
  - "The run log" is the run's own record (`State.Record`): the entry's stage
    is `reservation_renewed`, and `reservation_renew_failed` for a renewal
    that failed, since a run need not belong to a joined session of the
    shared run log. A step that renewed, or failed to, writes its state even
    when its stage waits or is refused, and the renewal's note is added to
    the step's next move or to the refusal's remedy.
  - The fence's refusals the table had no row for: the reservation gone from
    `origin`, unreadable, lapsed, released by the default branch (the record
    finished there), and an overlap met at the renewal, each contention; any
    other refusal of the renewal (no username) is an exit-2 refusal naming
    it. A fetch that fails at the fence is contention, not an outage. The
    fence runs where the pull request is left open for a person too, since it
    runs before the rule decides, and the arming's note names the renewed
    expiry.
  - The queue's timeout is the longest `check_response_timeout_minutes` a
    merge-queue rule on the default branch states in the mirror at the
    lane's base; the mirror here states 90, not 45.
- **What step 9 settled** (the tidy-up after the merge, 2026-10-14):
  - Only the landing of the lane that completes the run, with no step
    pending and every other lane done, tombstones the run's reservation: an
    intent's earlier lanes land while their siblings still need it.
  - The tidy-up is the landing's last step itself, after the worktree and
    the lane branch are removed, so the drain's finishing of merged lanes
    tombstones too, and a killed step that repeats it finds the tombstone and
    says it was already released.
  - Its outcome is a clause of the landing's record note: the tombstone
    pushed, the reservation already released, left as it stands with its
    holder named (another person, or your copy on another computer), left
    with the refusal named (absent or unreadable), or a fetch or push that
    failed, which says the reservation counts as released while the record
    is finished on the default branch. `claim.PushError` and
    `claim.FetchError` tell a failure reaching origin from a refusal judged
    on the tip.
- **What step 14 settled** (the drain's reservation per issue, 2026-10-14):
  - The drain reserves only in a checkout with a remote named `origin`; with
    none there is nowhere to reserve, the lane opens as before, nothing is
    handed back for want of a `found_at`, and the run records no
    reservation, as `abcd <record-id>` shows no line there.
  - The footprint is the path `found_at` names as the capture verb reads a
    path from it (`capture.FoundAtPath`): a `:line` locator dropped, then
    cleaned and held to `fsutil.ValidRelPath`. A `found_at` that is empty, a
    conceptual location in words, or not a repository-relative path is the
    hand-back.
  - The `found_at` hand-back is made before the host judgement is asked over
    the issue, so no judgement is spent on an issue that cannot be reserved.
    It is a flag routed home to the issue's record (`from` `reservation`,
    kind `no-found-at`), kept in the drain's hand-backs so the same drain
    does not meet it again.
  - The reservation is taken inside the run's start, after its checks and the
    budget check pass and before the run is created (an unexported start
    option carries the footprint, the seam step 13 fills from the spec), and
    the run's record names it (stage `reservation`). A start that then does
    not create its run releases a reservation it made fresh (reserved or
    taken over), never one this copy already held and renewed, nor one a
    racing start's run in this checkout now holds.
  - Another's hold (live, lapsed under the margin, unreadable within its
    hold, an overlapping footprint, a lost race) passes over the issue with
    the check `reservation`; every other refusal (no username, no local tier,
    `origin` not answering) refuses the drain's move, since it would refuse
    every issue alike.
  - A lane handed back has its reservation released: the drain pushes the
    tombstone as it routes the lane's hand-back, and the route's `wrote`
    names the outcome (released, left as it stands with its holder, or a
    push that failed, which leaves it to lapse). This line first left it to
    lapse with its lease; the review of 2026-10-14 had it released, since it
    held every other person off for up to two hours. A run `abcd build
    <iss-N>` starts holds no reservation until step 13, which must release
    its hand-back too, outside the drain.
- **A claim dated ahead of the reader** (review of steps 1 to 3, 2026-10-12)
  is clamped from the reader's clock as well as its own: `lease.Judge` takes
  the effective expiry as `min(ExpiresAt, min(ClaimedAt, now) + Max)`, and an
  unreadable stamp's hold as `min(WrittenAt, now) + grace`. This strengthens
  "neither a far-ahead clock nor a hand-edited expiry can hold a record past a
  day": no single reading promises more than a day from the reader's now, so
  a laptop whose clock reads next year shows a day's hold, not a year's. The
  clamp is applied at every reading, so such a claim keeps holding while its
  writer's clock stays ahead; the in-run claim shares the judgement and its
  tests pass unchanged.

## Open point

_None open._ Two readings are worth the facilitator's eye at review, and
neither blocks the build: A sibling worktree of one clone is its own checkout,
so it is "another copy" and meets the confirmed message "on another computer";
and the cap and the overlap are judged on the view fetched just before the push,
as scope condition 9 says.

## Records to touch

Assigned to the step that makes each true; the last step of each spec carries
the cross-cutting ones.

| Record | Step |
|---|---|
| `commands/reserve.md` (new), with the push as its documented meaning | 5 |
| `.abcd/development/brief/04-surfaces/37-reserve.md` (new; 36 is the dashboard's) and its row and help-group entry in `04-surfaces/README.md` | 5 |
| Generated `docs/reference/cli/commands.md` | 5, 6, 13 |
| `commands/peers.md` and `04-surfaces/08-abcd.md`'s peers section: the fetch and the reservation rows | 6 |
| `.githooks/pre-push` header: the exemption and its reason | 4 |
| A reservation-push `known_good` in `internal/core/guard/defaults/guard.json` | 4 |
| Brief invariant 10 gains one clause: A git branch push is not remote configuration (decision 12) | 7 |
| Brief invariant 7's documented-meaning entries for `reserve` and `peers` | 7 |
| `AGENTS.md` concurrent-sessions (reserve before starting; `abcd peers` lists reservations) and attribution (`Assisted-by: abcd:<version>` on reservation commits) | 7 |
| `docs/reference/terminology.md` gains "reservation" | 7 |
| `.abcd/development/brief/06-delivery/02-verification-matrix.md` rows for spec 1 | 7 |
| `commands/implement.md` and `04-surfaces/27-implement.md`: renewal, the fence and the tidy-up | 8, 9 |
| `commands/abcd.md` and `04-surfaces/08-abcd.md`: the reservation line and the view's age | 11 |
| `commands/build.md` and `04-surfaces/34-build.md`: the exclusion and the `peers` row, then `--release`, `--take-over` and the reservation at start, with the in-run claim within one account and the reservation across people | 12, 13 |
| `commands/drain.md` and `04-surfaces/35-drain.md` | 14 |
| Brief invariant 7's entries for `build`, `drain` and `implement step` | 15 |
| `AGENTS.md` concurrent-sessions: a build and a drain reserve as they start | 15 |
| `.abcd/development/brief/06-delivery/02-verification-matrix.md` rows for spec 2 | 15 |
| `ACKNOWLEDGEMENTS.md`, for any source the build adopts (none is planned: the primitive is git itself, decision 23) | 15 |
| The drafts index, `.abcd/development/brief/06-delivery/03-out-of-scope.md`, no longer lists this record | Removed in the change that wrote this spec, when the intent left `drafts/` |
| adr-2610031352212697 moves to accepted | 15 |

## Footprint

- packages: internal/core/lease, internal/core/claim, internal/core/implement, internal/core/implement/loop, internal/core/peers, internal/core/record, internal/core/site, internal/gitutil, internal/surface/cli, internal/core/guard/defaults, .githooks, commands/, AGENTS.md, docs/reference, .abcd/development/brief/02-constraints, .abcd/development/brief/04-surfaces, .abcd/development/brief/06-delivery
- tests: the shared lease judgement and the in-run claim tests unchanged; the message's format, size cap, empty tree and hostile values; the holder from a no-reply address, from the machine config and never from the repository; reserve, renew, take over, release and tombstone against a bare origin with clones for a second person, a second computer and a cloud session; the lost race; missing, rejecting and wrongly named remotes; the derived release; the pre-push exemption, its gate on a non-empty tree and its non-fast-forward refusal; the attribution gate over reservation commits; the peers listing; the cap and the overlap; renewal, the land fence and the tidy-up in `implement step`; the build and drain reservations; disk-only renders under the zero-network harness

## Steps

1. Spec 1: The shared liveness judgement
   - criteria: C10, C32
   - packages: internal/core/lease, internal/core/implement
   - tests: `lease.Judge` over a live, an expired, a clamped thirty-day and an unreadable stamp within and past its grace; `implement.DefaultLease`, `MinLease` and `MaxLease` equal the lease package's; the in-run claim judges through `lease.Judge`; every existing `implement` claim test passes unchanged
2. Spec 1: The reservation message, the holder and the copy key
   - criteria: C11, C12, C14, C17, C18, C19, C28, C29, C30, C31
   - packages: internal/core/claim, internal/core/site, internal/gitutil
   - tests: the message's trailers, subject and size cap; a message over the cap never written and a tip over it unreadable; a hostile tip read as quoted data with an unknown key ignored and a malformed time unreadable; a footprint with shell metacharacters, a colon and an asterisk stored verbatim; an invalid id refused before any ref name; a lease over a day refused; the holder from both no-reply forms, from `~/.abcd/config.json`, and refused when only the repository names one; the copy key minted once, exclusively, at `0600`, and refused without a local tier; no host, home path, session URL or tool footer in a message; `gitutil.EmptyTree` under SHA-1 and SHA-256
3. Spec 1: Reserve, renew, take over and release against the remote
   - criteria: C1, C2, C3, C4, C5, C6, C7, C8, C9, C13, C15, C20, C21, C22, C23, C52
   - packages: internal/core/claim, internal/core/peers, internal/gitutil, internal/core/implement/loop
   - tests: a bare origin and clones for P, Q, P's second computer and P's cloud session, with an injected clock; the parentless create leaving HEAD, index and `git status --porcelain` unchanged on a detached HEAD; every row of the push table; the margin on both sides; the unreadable hold and its takeover; the lost race through both git's rejection of an advertised tip and the remote's refusal of a push that landed while this one was received; the derived release from `resolved/` on `origin/main`; `upstream`-only, missing-path and `pre-receive`-rejecting remotes each making no reservation; `gitutil.Net` serving the land stage's existing tests unchanged
4. Spec 1: The pre-push exemption and the registry
   - criteria: C24, C25, C26, C33
   - packages: .githooks, internal/core/guard/defaults, internal/surface/cli, internal/core/claim
   - tests: the committed hook passing a reserve, a renewal, a takeover and a release with no receipt and no `--no-verify`; gating a push under the prefix that carries a non-empty tree, on a create and on an update; refusing a non-fast-forward to a reservation branch; the registry's fixtures passing the reservation push; `scripts/check-attribution.sh commits` passing reservation commits authored by a no-reply identity
   - records: the hook's header; the `known_good` entry
5. Spec 1: `abcd reserve` on the CLI and the plugin surface
   - criteria: C27, C28, C52
   - packages: internal/surface/cli, commands/, .abcd/development/brief/04-surfaces, docs/reference
   - tests: the verb's flags, usage errors and exit codes (0, 2 and 3); `--json` for each act; no footprint refused; take-over naming the copy replaced; renders carrying no host or home path; the surface-parity, help-group, sentence and command-reference tests regenerated and passing
   - records: `commands/reserve.md`; `04-surfaces/37-reserve.md` and its register row
6. Spec 1: `abcd peers` lists the reservations
   - criteria: C12, C16
   - packages: internal/surface/cli, internal/core/claim, commands/, .abcd/development/brief/04-surfaces
   - tests: the fetch with pruning and the listing of two reservations with holder, footprint, effective expiry, state and a live count of two; a hostile tip shown quoted and unreadable; a failed fetch rendering the last view with its age and exiting 1; the worktree rows unchanged; the fetched-at stamp written
   - records: `commands/peers.md`; the peers section of `04-surfaces/08-abcd.md`
7. Spec 1: The spec-1 records
   - packages: AGENTS.md, docs/reference, .abcd/development/brief/02-constraints, .abcd/development/brief/06-delivery
   - tests: `abcd lint`, `make record-lint` and `abcd lint docs` clean
   - records: invariant 10's clause; invariant 7's `reserve` and `peers` entries; AGENTS.md concurrent-sessions and attribution; "reservation" in terminology; the verification matrix's spec-1 rows
8. Spec 2: Renewal and the land fence in `implement step`
   - criteria: C40, C41, C42, C43, C44, C53
   - packages: internal/core/implement/loop, internal/core/claim, commands/, .abcd/development/brief/04-surfaces
   - tests: a step at half-lease renewing and logging it; one under half pushing nothing; the fence refusing a tip now another person's and one moved to the other copy, arming nothing; the fence renewing past a mirror's 45-minute timeout plus the margin, and with the default lease where no mirror exists; a renewal failure outside land named and not stopping the stage
   - records: `commands/implement.md` and `04-surfaces/27-implement.md`
9. Spec 2: The tidy-up after the merge
   - criteria: C45, C46, C51
   - packages: internal/core/implement/loop, commands/
   - tests: a landed lane tombstoning its own reservation; one leaving another person's and naming them; a failed tombstone push not undoing the landing; the lane branch removal leaving `abcd-reserved/<id>` untouched locally and on origin
10. Spec 2: The cap and the footprint overlap
   - criteria: C47, C48
   - packages: internal/core/claim, internal/surface/cli, commands/
   - tests: an eleventh live reservation refused naming the cap and the ten; a renewal and a take-over of one's own other copy never refused by the cap; an overlapping footprint refused with the back-off exit naming the other reservation and the shared path, for an equal path and for a parent directory; released and expired reservations counted by neither
   - records: `commands/reserve.md` gains the cap and the overlap
11. Spec 2: The record and board renders
   - criteria: C49, C50
   - packages: internal/core/record, internal/core/implement/loop, internal/surface/cli, commands/, .abcd/development/brief/04-surfaces
   - tests: `abcd <record-id>` and the board under the zero-network harness showing the reservation and the view's age; a never-fetched view saying so; the site's status block carrying no reservation
   - records: `commands/abcd.md` and `04-surfaces/08-abcd.md`
12. Spec 2: `build next` and `build <itd-N>` honour another person's reservation
   - criteria: C38, C39, C49
   - packages: internal/core/implement/loop, internal/surface/cli, commands/
   - tests: `build next` excluding the intent with the holder, the expiry and the view's age; `build <itd-N>` refused at the `peers` check naming the holder; the run's own reservation never a holder; both verbs added to the zero-network test
   - records: `commands/build.md` (the exclusion and the `peers` row)
13. Spec 2: `abcd build` reserves, releases and takes over
   - criteria: C34, C35, C36
   - packages: internal/core/implement/loop, internal/surface/cli, commands/, .abcd/development/brief/04-surfaces, docs/reference
   - tests: a new run and a resumed one reserving with the spec's footprint; a spec with no footprint refused at the reservation, starting nothing; `--release` pushing a tombstone and leaving the run's state; `--take-over` passed through; `build next` reserving the picked intent
   - records: `commands/build.md` and `04-surfaces/34-build.md`, the in-run claim within one account and the reservation across people
14. Spec 2: `abcd drain` reserves each issue
   - criteria: C37
   - packages: internal/core/implement/loop, internal/surface/cli, commands/, .abcd/development/brief/04-surfaces
   - tests: a drained issue reserved with its `found_at` path; an issue with no `found_at` handed back with the reason while the drain moves on
   - records: `commands/drain.md` and `04-surfaces/35-drain.md`
15. Spec 2: The ship records
   - packages: AGENTS.md, ACKNOWLEDGEMENTS.md, .abcd/development/brief/02-constraints, .abcd/development/brief/06-delivery, .abcd/development/decisions/adrs
   - tests: `abcd lint`, `make record-lint` and `abcd lint docs` clean; `abcd docs fidelity` review recorded before the close that ships the intent
   - records: invariant 7's `build`, `drain` and `implement step` entries; AGENTS.md concurrent-sessions for build and drain; the verification matrix's spec-2 rows; ACKNOWLEDGEMENTS if a source is adopted; adr-2610031352212697 to accepted
