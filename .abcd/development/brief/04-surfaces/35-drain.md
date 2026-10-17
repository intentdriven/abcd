# `/abcd:drain` — Sort the Open Ledger by What Needs No Decision

`/abcd:drain` is the verb a person types to have the open issue ledger worked
unattended: the issues that need no decision are fixed, and the rest are handed
back to the place a person decides them (itd-82, spc-2609212015054359). This
chapter describes what ships: the rule that decides which issues a machine may
take alone, the order it takes them in, a dry run that shows every open issue's
disposition and writes nothing, and the run, which asks the host to judge each
eligible issue's remedy before its lane opens, hands each issue the judgement
does not hand back to the implement loop keyed by the issue, one lane at a
time, routes every hand-back by its kind, and is bounded by the pace rule's
window and a cap on the lanes it opens.

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

## The rule

Which issues a machine may take alone is the drained repository's own decision
(the product thinker's ruling BX2 of 2026-09-29: "the PROJECT MUST HOLD the
eligibility decision in its own record (e.g. added at setup); drain refuses there
until it does"). The rule is read from an accepted decision record in the
repository's own store, `.abcd/development/decisions/adrs/`, whose frontmatter
carries four fields:

```yaml
drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]
drain_severities: [nitpick, minor]
drain_security: handback
drain_remedy: required
```

Those values are abcd's strict baseline, bundled in the binary as the measure a
repository's record is judged against. abcd's own repository states exactly the
baseline in
[adr-2609291342092738](../../decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when.md),
held there by invariant 19 in
[`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md) and a test.
The setup verb offers the baseline to a repository without a record and
writes it only on the person's yes (see [`01-ahoy.md`](01-ahoy.md)).

A repository's record may narrow the fixable set and the severities. It may
loosen abcd's floors (ruling H11 of 2026-09-29: "MAY LOOSEN abcd's floors (a
project may let drain take major/critical and security issues)"): listing
`major` or `critical` in `drain_severities`, or setting `drain_security: take`.
It may not widen `drain_categories` past the fixable set, since the other
categories are decisions by kind, and `drain_remedy` has the one value
`required`, since the remedy is the brief a lane works from. A record that
misses a field, misspells one, states one twice, lists a value the field does
not take, or writes a list as anything but an inline `[a, b]` refuses; so does a
store with two accepted records carrying the fields. None of these falls back
to the baseline or to a looser rule.

The rule reads each issue's fields and nothing else: nothing open in
`blocked_by`, no intent in `related_intents` (a consistency finding's aside,
below), a category the rule takes, a
severity it takes, and a `remedy:` other than `none (filed automatically)`, the value an automatic filer writes
when it has no fix. Every open issue receives exactly one disposition, from the
first rule that excludes it: skipped when blocked; handed back as `promoted`
when its `related_intents` names an intent, for `security`
(unless the record takes it), for a category outside the rule's set, for a
severity outside it, for a remedy that waits on a ruling, and for a deferral
that is live; ineligible without a remedy or with the automatic filers' value until
a person writes one; and unreadable when the ledger reader refuses the record. A
record written before the field existed reads its `suggested_fix:` as its remedy.

One automatic filer writes a real remedy. An after-merge fidelity audit that
judges a criterion `NOT_MET` or `INCONCLUSIVE` captures one issue carrying the
check it leaves owed (ruling DQ1c; [`05-intent.md`](05-intent.md)), and its
remedy is the work that clears it: "fix, then re-run the audit" for a failed
criterion, "re-run the audit" for an undecided one. The rule judges such a
record by its fields as it judges any other, and the record names the audited
intent in `related_intents`, so it is handed back as `promoted` whatever its
severity: a failed audit's `major` `bug` and an undecided one's `minor`
`inconsistency` alike are the intent's, paid through a re-run of its audit. A
person's own build of the issue is let through the hand-back, the eligibility
row naming the intent, when no other rule would hand it back: the rule is asked
again with `promoted` set aside, so a failed audit's `major` issue is still
refused on its severity, and the undecided audit's `minor` one is the lane
route its reopened work takes (ruling DQ1a, which DQ1c subsumes;
dcn-2610161519047899).

Three hand-backs hold whatever the repository's record says. An issue whose
`related_intents` names an intent is handed back under `promoted`, naming each
intent: an intent already owns it, so a machine never takes it alone, and a
promotion one drain made is never judged or worked again by another that reads
the promoted record. A record the consistency pass files is the one exception:
the two intents it names are the ones its contradiction is between, not intents
that took it on, so it is judged by its other fields, and is ineligible, as any
automatic filer's record is, until a person writes its remedy (ruling H12).
An intent it was promoted into, one that names it back in `related_issues`,
still owns it, and that intent is read from the same commit as the ledger: a
plan read from origin's default branch reads origin's intent store, never the
checkout's, so an intent origin holds owns its finding in a checkout that lacks
it, and a draft half-written in the checkout refuses no plan (a store that
cannot be read there refuses it, dcn-2610161835529757). The
other two each mark a decision a person still owes. A
remedy that opens "Waits on" as words, followed by a blank, a colon or nothing (the shape a remedy takes when its fix waits on an
unanswered ruling, compared case-folded), is
handed back under `waits-on-ruling`: taking it would make the ruling. A record
whose `deferred_after` names the checkout's current anchor tag, the newest
release tag, is handed back under `deferred`: a person carried it past this
release. A deferral past an earlier tag has lapsed and holds nothing back. A
record carrying both is named for the ruling, which says which decision is
owed.
The deferral verb writes a deferral only onto a `major` or `critical` record, but
a hand-written one on a lighter record is read the same way. The release tags are
read only when an open record carries a deferral, and not knowing whether a
deferral is live never lets its record through: a failure to read the tags
refuses the dry run, and a checkout holding no release tag (a shallow clone
fetches none) marks the anchor unknown and hands back every record carrying a
deferral, the dry run naming the missing tags and `git fetch --tags`. A checkout
holding only an older release tag (one not fetched since the last cut) is
caught the same way without asking a remote: a `deferred_after` newer than the
checkout's own tag, compared by core version, names a tag the checkout lacks, so
the anchor is stale. Every record deferred past a tag the checkout lacks is
handed back as `anchor stale`, naming that tag and `git fetch --tags`, and the
dry run names the stale anchor above them. A deferral past the local tag is
still handed back as live, since a tag named only in the ledger is not one the
checkout holds; it lapses when the newer tag is fetched. A `deferred_after` that is not a
release tag (`vMAJOR.MINOR.PATCH`) cannot be compared, and its record is handed
back too.

The fields are the rule because a model's judgement of its own ambiguity is
unreliable, and the failure runs one way: a machine that decides a thing needs
no decision, and then makes one. The host judgement over an eligible remedy is
therefore allowed only to hand an issue back (below); it does not run in the
dry run, and the dry run says so beside every eligible issue.

## The order

Eligible issues are taken by category in the order `tech-debt`,
`documentation`, `inconsistency`, `drift`, `bug`, `ux` (clean-ups and text
first, on the evidence that they merge most often), then `security` when the
record takes it; then by severity, `nitpick` before `minor` before `major`
before `critical`, as far as the record takes them. Within each group of one
category and one severity, the issues whose `found_at` names a file come before
those naming a directory or no path, and each part is taken oldest first in a
checkout with no remote named `origin`. Whether a path is a file is asked of the
commit the ledger was read from, each path on its own; a path that commit does
not hold (deleted since, or never there) is not a file, and sorts with the
directories, as do a submodule's gitlink and a path through a symbolic link. A
symbolic link itself is a file wherever it points, since git stores it as a
blob, and a checkout's working tree, read where there is no commit to ask, is
classified the same way, following no link. A
drain on `origin` takes each part in its own order instead: a rendezvous
(highest random weight) hash of the holder git's identity names with no network
act, the copy's key and the issue id, the first eight bytes of their sha256 read
as a number, heaviest first (`spreadWeight` in
`internal/core/capture/eligible.go`; dcn-2610161040368890). When git's identity
names no holder, that part of the key is empty and the copy's key alone tells
copies apart. Each account, and each of one person's copies, keeps one order
from move to move, and two of them start a group at different issues, so drains
on several accounts spread across it with no coordinator. The copy's key is
minted at its first move, before that move plans, so the order the first move
asks a judgement in is the one the next keeps. The record's lists are sets: the
order is abcd's. The dry run states the order and lists the eligible issues
first in it, then every other open issue by id; it mints no key, so on a copy
that has none yet it takes each group oldest first and says, in `order` and in
`order_drawn_at_first_move`, that the copy's own order is drawn at its first
move.

## Loud loosening

Every floor a repository's record loosens is named, measured against the
baseline, in this order: `severity major`, `severity critical`, `security`. The
dry run's text prints a `LOOSENED` block under the rule's record, or one line
saying the rule loosens none of abcd's floors; the machine-readable payload carries the list as
`loosened` beside the record's own values under `rule`; both modes also print a
warning on stderr naming each loosened floor; and the run's summary and stderr
name them at every move.

## The run

The run is driven by the host session, as the implement loop is (decision 5 on
itd-2609201916151817): each invocation of the bare verb performs one move and
exits. It reads the ledger afresh (the classification is re-derived every move
and written nowhere, decision 8), then does the first of these that applies.

In a checkout with a remote named `origin`, the ledger a move reads is origin's
default branch, never the checkout's own, which may be behind it
(spc-2610161044461448 step 1, dcn-2610161040350256): under the drain's lock,
before any judgement or reservation, the move fetches the reservations and that
branch, and reads one snapshot of the ledger out of the commit it fetched. The
plan, the lane's start check, the judgement's remedy digest and the
reservation's footprint all read that snapshot, so an issue another account
finished on origin is neither reserved, judged nor opened, and one filed there
and not yet in this checkout is judged and opened from origin's record. A move
that cannot read origin (the fetch fails, the fetch cannot read the default
branch, or no `origin/HEAD` names one) plans nothing: it records the failure in
the drain's state as an origin probe and waits as the outage gate waits, on the
shared schedule (one minute, five, ten, then hourly), a move before the next
probe reading nothing. It gives up as that gate does: a probe failing eight
hours into the hourly stage ends the drain with the stop reason `origin`,
saying what failed, since when, and that the next invocation begins a new
drain, which waits again if origin still cannot be read. It never plans from what the checkout last saw. The dry
run fetches nothing: it plans from origin's default branch as last fetched and
names the view's age, the commits the checkout is behind it, and the issues open
in the checkout that branch has finished. Any reservation of a record finished
on origin's default branch backs off, with a tip or without one
([`37-reserve.md`](37-reserve.md)).

1. When the implement loop's run has given up on a lost connection (the
   shared outage's eight hours of hourly probes have failed), it opens nothing
   and ends with the stop reason `outage`, naming the services down, since when,
   the probes made and the lanes it opened; once the outage is cleared, the next
   invocation begins a new drain.
2. Before the drain's `next_eligible_at`, it opens nothing and names the time.
3. When the drain's window has run its working minutes, it writes
   `next_eligible_at` (now plus the pause) into the drain's state and opens
   nothing; the next invocation after that time opens the next window and
   continues.
4. When the lane it opened last is still in progress, it names the run to drive
   with the implement loop's step verb and opens nothing: one lane at a time.
   A lane that cannot finish is handed back as stuck instead
   (spc-2610161044461448 step 5, dcn-2610161040382067). Each refused step of
   an issue run is logged on its lane with its stage, its check, its reason
   and a count of the same refusal in a row, and the drain reads the lane as
   stuck once the step has been refused the same way three times, or has kept
   being refused at one stage for sixty minutes, counted from its first
   refusal to its last, a quiet of more than fifteen minutes starting the time
   afresh (dcn-2610170118506273); both bounds are read from the
   pace's layers (`pace.stuck_refusals`, `pace.stuck_minutes`) at every move.
   A refusal that cannot change makes the lane stuck at once: the issue
   finished at the lane's base, or on origin's default branch as the land
   fence reads it, or the reservation now another person's. A wait by design
   is no refusal and is never counted: the landing's wait for its preflight
   receipt, every outage wait, and a land fence that cannot read origin
   (dcn-2610170116582725). The drain hands a stuck lane back as
   `stuck`, naming the refusal, releases its reservation with the hand-back's
   tombstone, and goes on to the next issue in the same move. The lane keeps
   its worktree and branch until the person discards it through the step verb,
   which ends the run.
5. It reads what that lane has come to: its pull request opened (armed, or left
   open where the repository has no merge queue), the run complete, or the lane
   handed back, which it routes (below). A pull request the forge reports
   conflicting is refused before it is armed, so its lane stays in progress
   and the drain names its run to drive, whose step names the conflict; one
   that turns conflicting after it was armed is refused by the run's own step,
   since the drain's part ends at the open pull request.
6. When the drain has opened as many lanes as its cap, it ends and says so.
7. For the next eligible issue in the drain order that the drain has not taken
   or handed back and this checkout has no run for, it asks the host judgement
   over the issue's remedy (below) and opens nothing, unless the host has
   already answered no over the remedy as it stands. With that no, it starts
   the implement loop for the issue and names the run. An issue the loop's own
   checks refuse (a peer holds it) is passed over, named with the check. When
   none is left, held or not, the drain ends and says so. At its holder's cap
   of live reservations, counted in the move's view before any reservation or
   judgement, or when every eligible issue left is held by another, it neither
   refuses nor ends (spc-2610161044461448 step 6, dcn-2610161040364312): it
   reserves and asks nothing, writes `waiting_until` fifteen minutes on, kept
   apart from `next_eligible_at` so the pace window is not reset, names what it
   waits on (the reservations and the pull requests waiting to merge, with the
   advice to wait for a merge rather than release one, or the holds), and keeps
   its judgements. A move before that time still looks, since passing over held
   work fetches and pushes nothing, and one that still waits keeps the time
   first written. In a checkout with a remote named
   `origin`, the issue is first reserved there before its judgement is asked,
   with the path its `found_at` names as the footprint (itd-2609091034175565,
   C37; spc-2610161044461448 step 2), so the reservation push is the
   compare-and-swap between accounts: while the drain's reservation holds, no
   other account's drain reserves, judges or opens the issue, and the lane a no
   opens starts on it with no second push. The judgement awaited records the
   commit that reservation made (`judging.hold`), and the drain releases that
   commit alone, by a tombstone, on every path the process lives through that
   leaves it unused, a panic included: a yes; an
   answer over an issue that has left the eligible set; a judgement superseded (the order
   now asking another issue, nothing eligible left, the drain stopping at its
   cap, an outage or an origin give-up, or a no answered once the window has
   elapsed); a lane's start refused or failing; and a request that lapsed.
   Each is a fetch and one push, made only while that commit is still the tip,
   so a reservation this person made or renewed by hand since is left as it
   stands. A process killed outright (SIGINT or SIGKILL) runs no release, so
   its hold lapses after one lease. Every push the drain makes for a judgement
   is its hold, a renewal of this person's reservation that had lapsed
   included; one this person still held live when the drain reached the issue
   is theirs and never released. When the move can read origin's
   reservations, a recorded hold is asked on only while the view still shows
   its commit at the tip; otherwise the drain reserves again, and records that,
   before it asks. When it cannot read them, the move says so and asks on the
   recorded hold, and every later release or lane start is still checked
   against the tip.
   A yes's tombstone publishes the hand-back on origin
   (spc-2610161044461448 step 3, dcn-2610161040365515): it names the yes's
   kind (`Abcd-Handed-Back`) and the blob id of the issue's record on origin's
   default branch (`Abcd-Record`), the closed form
   [`37-reserve.md`](37-reserve.md) states, so no account's drain reserves,
   judges or opens the issue again until a person changes that record there:
   a person who reserves the issue over it and releases it again without
   changing the record, by hand or through a `build` start that fails, leaves
   the hand-back standing, since that release's tombstone keeps it.
   A user-visible yes over an issue this checkout does not hold as origin's
   default branch does (absent here, or held here as another record, judged
   by git's object name for this checkout's file) promotes nothing here: the
   route is a flag whose home is a checkout that holds the record, and the
   hand-back is published all the same, so the promotion is left to a person
   whose checkout can make it. A yes whose hold the move cannot release with
   the hand-back on it (renewed or reserved again by hand, taken over,
   released by a person, or never taken because this person held the issue
   live from this copy), or whose release did not reach origin, says in its
   route that the hand-back is not yet published, and why; the drain keeps it
   pending in its state, carries it into the next drain, and publishes it over
   the issue's next release on origin, a tombstone marking no hand-back, as
   that tombstone's child, fenced on it and on the blob the yes was judged
   over; a tip already marking a hand-back, a record finished or changed on
   origin's default branch since, one origin holds in no one file, or a
   tombstone the hand-back would take past its cap ends it with nothing
   published, and the move says it was dropped and why; a default branch that
   could not be read keeps it pending (dcn-2610161826443524, refined by
   dcn-2610161931130911). The
   hold is taken once per request and never renewed by a poll: a request whose
   reservation lapses unanswered is dropped, and its reservation released at
   once, before anything else in that move. The issue is not asked again in
   that move, nor before a cool-down of the take-over margin (five minutes)
   from it, so another account has a window to take it: it is passed over
   meanwhile with the check `lapsed`, and a drain left with no other work
   waits rather than ending. It is then asked again, on a new reservation,
   only after the drain's other eligible work, and the lapse is forgotten once
   it is asked again or answered (dcn-2610161315385888, refined by
   dcn-2610161420314313). An issue whose `found_at` names no repository-relative path is
   handed back before any judgement over it, `from` `reservation`, with the
   reason "`<iss-N>` has no found_at path to reserve it with; handed back", and
   the drain moves on. An issue another copy holds in the move's view, or whose
   footprint shares a path with another copy's live reservation, is passed over
   with the check `reservation`, naming the holder, before anything is pushed
   or asked, so passing over held work adds no fetch to the move's one; one
   the reservation push itself finds held (a race since the fetch) is passed
   over the same way. One the person's own other copy holds live says so and
   names `abcd drain --take-over`; a move given that flag lets that copy's
   reservation of the issue through the view, and its reservation takes it over
   for this copy, as `abcd build --take-over` does, so a hosted session that
   restarted with its local tier lost takes its own reservations back as the
   drain reaches them (spc-2610161044461448 step 15, dcn-2610161040398791). The
   flag is the move's alone, never kept in the drain's state, and never moves
   another person's reservation. An issue whose tip in the move's view is a hand-back
   that stands, from any copy on any account, is passed over the same way,
   naming its kind, who handed it back and when, and every reservation the
   drain pushes backs off a hand-back that stands, so one the view missed is
   passed over by the push. A drain's push that would take over or renew a
   reservation a person made over a hand-back that stands backs off it too,
   lapsed or not: the push reads that reservation's line down to the
   tombstone it was made over (dcn-2610161931122606). A reservation refused for this checkout's own reason (no
   GitHub username, `origin` not answering, `origin` a fork the repository does
   not allow) refuses the move; the fork check is made by the reservation taken
   before the judgement, and one it could not make, or waived for the allowed
   fork, is named in the run record and start result of the lane a no opens on
   that reservation, in whichever move it opens, from the judgement that keeps
   the hold it was asked on; a lane whose start renews a judgement's hold that
   lapsed before the no, and a judgement's reservation that renews this copy's
   own lapsed one, ask again, the push making the hold the drain's afresh
   ([`37-reserve.md`](37-reserve.md), dcn-2610161827036320). A move refused
   once it has done anything (the judgement block onward: the outage read, the
   merged lanes, a lane's routing, the run listing, the origin check, a
   judgement's reservation or request, a lane's start) writes what it did
   first, and its refusal says whether the answer given was taken; the hold
   the written state no longer records is released (dcn-2610161826557910). A peer this checkout
   cannot read refuses the move once, where per-issue contention begins, after
   the move's housekeeping, which is written first, and before any issue is
   judged or reserved, naming each peer with the repair its reason calls for and
   never a removal (spc-2610161044461448 step 9, dcn-2610161040378713): what
   such a peer holds is unknown, so left to the peers check it would read as
   holding every issue. The run records the reservation, which its steps renew, fence
   and tombstone ([`27-implement.md`](27-implement.md)); every drain move
   renews, at half its lease, the reservation of each issue lane parked at
   land, so a pull request waiting on a person stays held while the drain
   moves, and the reservation of the lane in progress, before its window gate,
   so a move that waits renews too, whether or not a step of the lane ran
   (spc-2610161044461448 step 7, dcn-2610161040374343). The landing checks the
   reservation before its push as the land fence does before arming, and a
   lane whose hold is now another person's pushes nothing, opens no pull
   request and is handed back at once as stuck; a lane handed back has its reservation tombstoned by the step that
   hands it back, or by the drain's route when that step's push failed, the
   tombstone publishing the hand-back with the lane's kind (`unachievable` for
   a lane stopped after its fix rounds), and the route the drain writes names
   the outcome. A run `abcd build <iss-N>` started publishes its lane's
   hand-back the same way. Without `origin`
   nothing is reserved.

The host's answer is taken first of all, before the window clock and the
lane's routing: an invocation given the answer's path validates it and records
it, routes a yes, and then moves as above.

The drain's state is one file beside the runs,
`.abcd/.work.local/run/drain.json`: when it began, the rule's record, its cap
and its pace, the window clock, every lane it opened with its outcome, and
every hand-back it routed with the record change it made. It is written under
its own lock, so two drains never open two lanes. A drain that has ended is
kept for reading, and the next invocation begins a new one. The cap and the
pace are set when a drain begins; a different cap or pace named while it runs is
refused rather than ignored. The pace is the implement loop's, resolved through
the same layers, and each lane's run is paced as a run is.

### The judgement before a lane opens

The field rule decides what a machine may consider; one question is left that
no field answers, and it is the host's (adr-25): does this remedy, carried out
as written, change what a user sees, or a trust boundary? The drain asks it the
host-pass way. It writes a request into the local tier,
`.abcd/.work.local/run/drain-judgement.request.md`, carrying the question, what
each answer does, the issue's id, record path, severity and category, its
remedy and its record's body, each quoted between begin and end markers with
any comment marker inside escaped, as a lane's brief quotes them, and the
answer's exact shape; it records in its state the
issue, the request and the answer's path, and the sha256 of the remedy the
request shows, and opens nothing. One judgement is awaited at a time. A move
made while it is awaited asks again over the same issue and keeps any answer
already written; a move that asks about another issue, or about a remedy since
rewritten, first removes the earlier answer, so it can never be taken for this
one.

The host writes its answer to `.abcd/.work.local/run/drain-judgement.json` and
hands it back to the run, naming the answer's path. The answer is strict JSON:
`schema_version` 1, the `issue` and the `remedy_sha256` the request names,
`answer` `yes` or `no`, a `kind` with a yes (`user-visible`, or `trust-rule`,
which a remedy that changes both takes) and none with a no, and a `reason`; any
other field refuses it. The request tells the host to answer yes when the
remedy does not let it tell, since the failure the judgement guards runs one
way.

- A **no** changes nothing: the issue's lane opens in that same move, as its
  fields already allow, and is never asked about again in this drain while its
  remedy stands.
- A **yes** hands the issue back before any lane opens, routed exactly as a
  lane's hand-back of the same kind (the table below): a user-visible change is
  promoted to an intent draft, and a trust rule is flagged as needing a
  decision record with the host's reason as its question. The route is in the
  summary with `from` `judgement`, and the drain asks about the next issue.
- **It never lets an issue through.** The question is asked only of an issue
  the field rule found eligible in the same move. An answer over an issue that
  has left the eligible set since the request (its severity raised, a blocker
  filed, the record resolved) is recorded with a note saying why and decides
  nothing, whatever it says. An answer over a remedy rewritten since the
  request is refused, since it judges a remedy the lane would not work from,
  and the next move asks again.

Every answer is kept in the drain's state with whether it decided the issue's
disposition, and is in the summary in text and in the machine-readable payload;
nothing is written onto the issue for it (decision 8).

The judge is a role, `drain-judge`, on the runner routing the build roles take
(spc-2610161044461448 step 14, dcn-2610161040390001). With no route, the
default, the request above is the host's. When the person's own layer routes
`roles.drain-judge.runner` to a runner, the move that writes the host's request
also writes the runner a brief of its own,
`.abcd/.work.local/run/drain-judgement.runner.md`, the same question over the
same quoted record with an answer shape that carries no digest, and starts the
runner on it through the dispatch every routed role takes, granted Read and
Write alone and bounded at fifteen minutes. Its answer, at
`.abcd/.work.local/run/drain-judgement.runner.json`, is read as strictly as the
host's, abcd filling in the digest of the remedy the brief showed, and taken in
that same move as `--judgement` would take it: a no opens the lane on the
reservation the drain took to ask it, and a yes is routed and the drain asks
about the next issue, so a drain started in a plain terminal needs no person's
answer. The judgement records the route that ran it. A runner that does not
answer, refuses, fails, or writes an answer out of shape is a fallback: its
receipt is kept in the drain's state and the host's request stands, as with no
route; a rate limit leaves it standing too. The runner runs under the drain's
lock, and whatever it reaches over the network is the runner's own, not a
network act of abcd's.

### The lane

Each lane is the run `abcd build <iss-N>` starts (decision 10 on
itd-2609201916151817). Its checks are the rule above, read the same way, and the
peers check; the key is an issue id by shape before any path is built from it.
Its brief is the issue's record, read at the lane's base, with its remedy as the
work and the repository's definition of done: a detector watched to fail before
the fix and pass after. Its validators run without the fidelity audit, since an
issue has no criteria. Its implementer's receipt must name the issue in
`resolves`, and the landing resolves the issue with the commit named there in
the lane's own change, opens one pull request and pushes nothing after arming.

### The hand-back, by kind

A lane that finds a decision in its issue writes `handback` (a kind, a reason
and, where the kind needs one, a home) in its receipt instead of resolving it.
The loop reads it before the validators, discards the lane's worktree and
branch, records the discarded head, and ends the lane. The drain then routes it:

| Kind | Route | What is written |
| --- | --- | --- |
| `user-visible` | the issue is promoted to an intent draft by the capture verb's promotion | the draft, and the issue's `related_intents` naming it; nothing else |
| `trust-rule` | flagged as needing a decision record, the lane's reason as the question | nothing; no record is minted |
| `design-finding`, `second-package` | flagged with the home the lane names | nothing |
| a lane stopped after its fix rounds | flagged, the issue staying open with the last findings | nothing |
| a lane handed back as stuck (`stuck`) | flagged, the issue staying open, naming the discard of its run | nothing |

Whatever the route, the issue's reservation is released by a tombstone that
publishes the hand-back on origin, naming its kind (a lane stopped after its
fix rounds as `unachievable`, a stuck one as `stuck`) and the record's blob, which every drain passes
over until a person changes the record (above).

Every issue the rule itself hands back (by category, severity, security, a
ruling or a deferral) is flagged in the summary naming the rule. Every route is
in the summary, in text and in the machine-readable payload, with the record
change it made, so nothing is dropped silently.

### The summary's counts

Every move's summary totals the drain as it stands, in text and in the
machine-readable payload's `counts`: the lanes opened, by outcome (in progress,
handed back), and the pull requests they opened; the hand-backs routed, by
route (promoted, flagged for a decision record, flagged with a home); the
issues the rule flags; the host judgements taken; the eligible issues the move
passed over; and the spend. The spend is counted in agent runs, the unit the
budget check estimates a run in: each implementer's receipt the loop verified
across the drain's lanes (the implement stage's, each fix round's, a
hand-back's) and each validator's return it read a verdict from. No runner
reports tokens or money, so neither is counted. A lane whose run state cannot
be read is named in the spend, which is then a floor, rather than refusing the
summary. A merge is a person's gate, so the drain counts the pull requests its
lanes opened, not the ones merged; a drain that ends with every lane ended and
none in a pull request says so as a normal ending and exits 0, and one that
stops with a lane still in progress (an outage give-up) claims no such ending.

## The trust boundary

The eligibility record is a file the drained repository authors, and it decides
what an unattended agent may change there. The threat is a contributor's pull
request that loosens it, for instance adding `major` or `drain_security: take`,
so that a later drain takes issues a person would have decided. What guards it:

- The record is committed history in the decision store, reviewed like code, and
  a change to it is a change to a decision record, which a reviewer reads as a
  trust change.
- A loosening is loud: the dry run, its stderr and the start name every floor
  the record loosens, so a loosened rule is never applied unseen.
- abcd's own repository keeps the strict baseline, and a test fails when its
  record loosens anything or stops being the record the invariant cites.
- The reader never falls back: a missing, partial, ambiguous or malformed record
  refuses. A record stating any frontmatter key twice is malformed, because the
  line scanner keeps the first value and a YAML reader the last, so
  `status: accepted` then `status: superseded` would read as two decisions; so
  is a record whose frontmatter `id` disagrees with the id its file name gives
  it, which would put another record's name on its rule.
- The store is read inside the checkout and each record through the capped
  trust-boundary reader, so a store that is a symlink or sits below one,
  wherever it points, a record that is a symlink at all, and a record past the
  ledger's size cap are refused rather than followed or read whole. The store
  and its records are held to one rule: the decision record is the one
  committed at its own path, and one reached through a link, even a link inside
  the checkout, is not it.
- The host's judgement is read as untrusted input: only from the answer's
  path the drain named, as a regular file within 64 KiB through the capped
  trust-boundary reader, strictly decoded (no unknown or repeated field), and
  bound to the issue and the digest of the remedy the request showed; a
  refused value is described, never echoed, and the reason it carries is
  capped and sanitised. Whatever it says, it can only hand an issue back.
- A lane's hand-back is the implementer's word, read as untrusted input: its
  kind is one of the four the loop routes, its reason and home are present,
  capped and sanitised, and a hand-back beside a resolution is refused. The
  loop discards only the worktree it made at the lane's path on the lane's own
  branch, and deletes the branch only at the tip it read.
- The two person-owed hand-backs, a remedy waiting on a ruling and a live
  deferral, hold whatever the record says, and so does a deferral whose
  liveness the checkout cannot read: no release tag, a tag newer than the
  checkout's own, or a value that is not a release tag.

## What it refuses

The dry run and the bare verb both refuse, exit 2 with nothing written, when
the repository holds no accepted record of the rule, naming how to add one
(the setup verb's offer, or the four fields on an accepted record); when a record
names the fields but is proposed or superseded, the refusal names it. They
refuse a malformed record, naming the record and the field, two accepted
records, naming both, and a store or record that cannot be read safely. The run
also refuses a checkout without the local tier, a cap that is not a whole
number, a cap or pace other than the one a drain in progress began with, and a
drain state it cannot read as its own; another drain moving in the checkout is
a contention (exit 3). It refuses a judgement's answer when the drain awaits
none, when it is given at another path than the one the request names, when it
cannot be read or is not exactly the request's shape, and when the remedy it
judges has been rewritten since; each refusal writes nothing. The dry run
refuses the run's own flags. A run that
opens nothing or merges nothing exits 0 and says why. A checkout that cannot be resolved,
or a ledger holding one id in two status folders, is refused as every capture
verb refuses it.

## Where this sits

- The intent and its decisions: itd-82; the design record:
  spc-2609212015054359, which stays open for the decision-record marker and
  the issues its plan names.
- The lane it hands issues to: itd-2609201916151817 decision 10,
  [`34-build.md`](34-build.md) and [`27-implement.md`](27-implement.md).
- The field it reads is written by capture: [`06-capture.md`](06-capture.md).
- The plugin surface: `commands/drain.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd drain`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--dry-run` | bool |
| `--fix-rounds` | string |
| `--judgement` | string |
| `--max` | int |
| `--pace` | string |
| `--sub-agents` | string |
| `--take-over` | bool |

<!-- surface-appendix:end -->
