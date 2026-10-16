---
name: drain
description: "Fix the issues needing no decision, one lane at a time, and hand the rest back: Writes its state and user-visible drafts; refuses without the rule's record."
argument-hint: "[--dry-run] [--max <n>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>] [--judgement <file>]"
block: agents
user-invocable: false
---

# `/abcd:drain`

Work the open issue ledger: fix, one lane at a time, the issues this
repository's own rule says need no decision, and hand every other one back by
kind. The dry run shows what a drain would do and performs **zero writes**; the
run performs one move per invocation.

Show the plan first:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" drain --dry-run --json
```

Then run the drain, one move at a time:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" drain --json            # all eligible issues
"${CLAUDE_PLUGIN_ROOT}/abcd" drain --max 3 --json    # at most three lanes
```

## The rule

Which issues a drain may take alone is **this repository's own decision**: an
accepted decision record in `.abcd/development/decisions/adrs/` whose
frontmatter carries four fields. abcd's strict baseline, which
`abcd ahoy install` offers to write, is:

```yaml
drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]
drain_severities: [nitpick, minor]
drain_security: handback
drain_remedy: required
```

A repository's record may narrow those lists, and may **loosen abcd's floors**:
list `major` or `critical` in `drain_severities`, or set `drain_security: take`.
It cannot widen the categories past that fixable set, and `drain_remedy` is
always `required`.

An open issue is **eligible** when all of these hold, read from its fields
alone:

- no record in its `blocked_by` is still open;
- its `related_intents` names no intent (an issue linked to an intent goes
  through that intent, whatever the repository's record says), unless the
  consistency pass filed it: the two intents such a record names are the ones
  its contradiction is between, so once a person has written its remedy it is
  judged by the other fields below as if it named no intent (an intent it was
  promoted into, which names it back in `related_issues`, still owns it);
- its category is one the rule takes;
- its severity is one the rule takes;
- it carries a `remedy:` (`abcd capture --remedy "<fix>"`, which every new
  issue carries, or `abcd capture remedy <iss-N> "<fix>"` onto an open one; a
  record carrying only the older `suggested_fix:` reads that as its remedy),
  the remedy is not `none (filed automatically)`, the value abcd's automatic
  filers write when they have no fix, and it does not open "Waits on";
- it carries no deferral that is live at the checkout's newest release tag,
  none past a release tag newer than that one (a tag the checkout lacks), none
  that is not a release tag, and none at all when the checkout holds no release
  tag.

The rules are asked in a fixed order, and the first that excludes an issue
decides its one disposition:

| `outcome` | `rule` | Meaning |
| --- | --- | --- |
| `skipped` | `blocked` | an open record blocks it; `blockers` names them |
| `handback` | `promoted` | its `related_intents` names an intent, and the consistency pass did not file it; the reason names each intent once, at most three and then how many more, and the way through the intent: `abcd build <itd-N>`, or `abcd intent audit <itd-N>` for an audit's owed check |
| `handback` | `security` | category `security`, a person's unless the rule sets `drain_security: take` |
| `handback` | `category` | a category the rule does not take |
| `handback` | `severity` | a severity the rule does not take (`major` and `critical` under the baseline) |
| `handback` | `waits-on-ruling` | its remedy opens "Waits on": the fix waits on a ruling a person has not given |
| `handback` | `deferred` | its `deferred_after` names the current anchor tag: a person carried it past this release; or the checkout holds no release tag (a shallow clone fetches none), so whether any deferral is live is unknown and the reason names `git fetch --tags`; or it names a release tag newer than the checkout's own, which the checkout lacks, so the reason says `anchor stale` and names that tag and `git fetch --tags`; or it is not a release tag at all |
| `ineligible` | `remedy` | no remedy (a record filed before the remedy was required), or `none (filed automatically)` from an automatic filer; ineligible until a person writes one with `abcd capture remedy`, which the reason names |
| `unreadable` | `unreadable` | the ledger reader refuses the record; the reason names why |
| `eligible` | `fields` | every field rule passes |

The dry run stops at the fields. The run asks one more question of each
eligible issue before its lane opens, the host judgement over its remedy
(below), and that judgement can only ever hand an issue back.

## The order

Eligible issues come first, in the order a drain takes them: by category
`tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux`, then
`security` when the rule takes it; then by severity, `nitpick` first; then,
within each group of one category and severity, the issues whose `found_at`
names a file before those naming a directory or no path (a symbolic link is a
file wherever it points, as git stores it; a path origin's commit does not
hold, a submodule, and a path through a link count as no file, in the working
tree as on origin); then each
part oldest first without `origin`, or, on `origin`, in this drain's own order:
a rendezvous hash of the holder git's identity names with no network act, this
copy's key and the issue id, so every account and copy keeps one order from
move to move and two of them start a group at different issues. When git's
identity names no holder, the copy's key alone tells copies apart. The
payload's `order` states the rule. Every other open issue follows, by id.

## Loosened floors

When the repository's record loosens a floor, the text output prints a
`LOOSENED` block naming each one (`severity major`, `severity critical`,
`security`), stderr carries a warning naming them in both modes, and the
payload's `loosened` lists them. **Tell the user every loosened floor first,
before anything else in the plan**: a loosened rule lets a drain take issues
abcd's baseline hands to a person, and the record that loosened it is a change
a person should have reviewed.

## The payload

`dry_run` is `true`; `record` is the repository's decision record the rule is
read from; `rule` is that record's rule (`record`, `path`, `categories`,
`severities`, `security`, `remedy`, `loosened`); `loosened` lists every floor
it loosens (empty when none); `anchor` is the checkout's newest release tag, the
one a live deferral names, present when an open record carries a deferral; `anchor_unknown` is `true` when
an open record carries a deferral and the checkout holds no release tag;
`anchor_stale` names the newest release tag an open record is deferred past
that is newer than `anchor`, present when the checkout lacks it (every record
deferred past such a tag is then handed back, and a deferral past `anchor` is
still handed back as live, lapsing only when the newer tag is fetched); `order` is the ordering rule;
`dispositions` holds one entry per open issue (`id`, `path`, `title`, `severity`,
`category`, `outcome`, `rule`, `reason`, and `blockers` when skipped); `counts`
totals them by outcome; `ledger` names the checkout and branch read.
`order_drawn_at_first_move` is `true` when this copy has no key yet: the dry
run mints none, so it takes each group oldest first, and `order` says the
copy's own order within each group is drawn at its first move, which mints the
key. Once the copy has a key, the dry run shows the order its next move takes.

In a checkout with a remote named `origin`, the dry run plans from origin's
default branch as this checkout last fetched it, and fetches nothing: `source`
names that branch and its commit; `origin` carries `ref`, `commit`,
`fetched_at` (when abcd last fetched here, `null` when nothing recorded it),
`age_minutes`, `behind` (the commits of that branch this checkout lacks) and
`finished` (the issues open in this checkout that the branch holds resolved or
wontfix). When this checkout names no default branch of origin, `origin` is
`null`, `origin_unread` says why, and the plan is the checkout's own ledger.
Tell the user the view's age and how far behind the checkout is, and name each
finished issue: a drain will not take it.

Tell the user any loosened floors, then the counts, then the eligible issues in
order, then the others grouped by outcome with their reasons. For an
`ineligible` issue, say that a person writing a remedy with
`abcd capture remedy <iss-N> "<fix>"` is what makes it a candidate, and name the
ones an automatic filer wrote apart, since their reason says so. For a
`waits-on-ruling` or `deferred` hand-back, say which ruling or release it waits
on, and for a `promoted` one, which intent it goes through; a person's own
`abcd build <iss-N>` still takes a `promoted` issue no other rule would hand back, naming the intent. Do not act on the list: a hand-back is a person's decision.

## The run

Each `abcd drain` without `--dry-run` performs one move and exits 0, saying
what it did in `next`:

- **It reads the ledger from origin.** In a checkout with a remote named
  `origin`, every move first fetches the reservations and origin's default
  branch, under the drain's lock, and plans from the ledger that branch holds,
  never from the checkout, which may be behind: an issue another account
  finished there is not taken, and one filed there and not yet in this
  checkout is judged and opened from origin's record. `origin` names the
  branch, its commit, the commits this checkout is behind and the issues open
  here it has finished. A move that cannot read origin (the fetch fails, the
  default branch cannot be read, or no `origin/HEAD` names one) plans nothing:
  it asks no judgement, reserves nothing, records the failure in the drain's
  state (`origin_wait`: `since`, `kind` — `fetch`, `default-branch` or
  `origin-head` — `probes`, `next_probe_at`, and `hourly_since` once the
  probes are hourly) and says in `next` when it looks again, one minute, five,
  ten, then hourly; a move before then reads nothing. It gives up as the outage
  gate does: a probe that fails eight hours into the hourly stage ends the
  drain with `stopped: "origin"` and `complete: true`, and `next` says what
  failed, since when, the probes made and the lanes the drain opened; the
  next `abcd drain` begins a new drain, waiting again if origin still cannot
  be read. The
  text form reads "ended: the drain gave up waiting on origin".
  For `origin-head`, run `git remote set-head origin --auto` and
  `git fetch origin`. Tell the user, and run `abcd drain` again at or after
  `next_probe_at`.

- **It asks the host judgement.** Before the next eligible issue's lane opens,
  the drain writes a request, `.abcd/.work.local/run/drain-judgement.request.md`,
  and opens nothing: `judging` names the issue, the request, the answer's path
  and the digest of the remedy the request shows. Read the request and answer
  its one question yourself, from the request alone: does this remedy, carried
  out as written, change what a user sees, or a trust boundary? Write the JSON
  it asks for to the answer's path,
  `.abcd/.work.local/run/drain-judgement.json`, then run
  `abcd drain --judgement .abcd/.work.local/run/drain-judgement.json --json`.
  A `no` lets the lane open in that same move. A `yes` names its kind,
  `user-visible` or `trust-rule` (`trust-rule` when it is both), and hands the
  issue back before any lane opens, routed as a lane's hand-back of that kind
  (below), `from: "judgement"`; the drain then asks about the next eligible
  issue. When the remedy does not let you tell, answer `yes` and say what you
  could not tell. Do not open the lane, edit the record or fix anything while
  judging.
- **It reserves the issue.** In a checkout with a remote named `origin`, the
  drain reserves each issue on it before it asks the judgement over it, as
  `abcd reserve` does, with the path the issue's `found_at` names (a `:line`
  locator dropped) as the footprint, so every other person and computer sees it
  is taken: while it holds, no other account's drain reserves, judges or opens
  the issue. `judging.hold` records the reservation the drain took (`commit`,
  `claimed_at`, `expires_at`), and the lane a `no` opens starts on it, with no
  second push. The drain releases that reservation, and no other, whenever no
  lane will open on it: on a `yes`; on an answer over an issue that has left
  the eligible set; when the judgement is superseded (the order now asks another issue,
  nothing eligible is left, the drain stops at its cap, on an outage or giving
  up on origin, or a `no` comes once the window has elapsed); when the lane's
  start is refused or fails; and when a request lapses. It does so on every
  path the drain's process lives through, an error or a crash in the move
  included; a drain killed outright (Ctrl-C, or a kill signal) releases
  nothing, and its reservation lapses after one lease (two hours by default).
  Each release is a fetch and one tombstone push over the commit the drain's
  reservation made, made only while that commit is still the tip, so a
  reservation you made or renewed with `abcd reserve` since is left as it
  stands. A poll that can read origin's reservations checks the same tip
  before it asks again: when the reservation recorded is no longer there (you
  released it, say), the drain reserves the issue again first. A poll that
  cannot read them says so, and the release still checks the tip. A reservation of yours that is still live
  when the drain reaches the issue is yours, and the drain never releases it;
  one of yours that has lapsed, the drain's push renews, and it is the
  drain's from then on. A `yes` releases with a tombstone
  that publishes the hand-back on `origin`: its kind (`Abcd-Handed-Back`) and
  the blob id of the issue's record on origin's default branch
  (`Abcd-Record`), never free text. A user-visible `yes` over an issue this
  checkout does not hold as origin's default branch does (absent here, or
  held here as another record) promotes nothing here: the route is a flag
  whose `wrote` says why and whose `home` names the checkout that can promote
  it, and the hand-back is published all the same. The reservation is taken once
  per request and never renewed while you poll: a request still unanswered
  when it lapses (two hours by default) is dropped and its reservation
  released at once, in the first move after the lapse. That move does not ask
  the issue again, nor does any move within five minutes of it (the take-over
  margin), so another account has a window to take it: meanwhile the issue is
  passed over, check `lapsed`, naming when it is asked again, and a drain with
  no other work waits rather than ending. After that it is asked again, on a
  new reservation, only after the drain's other eligible work. An
  issue whose `found_at` names no repository-relative path cannot be reserved:
  it is handed back before any judgement is asked over it, `from:
  "reservation"`, with the reason `<iss-N> has no found_at path to reserve it
  with; handed back`, and the drain moves to the next issue. An issue another
  copy holds in the move's view, or whose footprint shares a path with another
  copy's live reservation, is passed over before anything is pushed or asked
  (`passed`, check `reservation`, naming the holder), with no fetch beyond the
  move's one; one the push itself finds held is passed over the same way. An
  issue whose tip on `origin` is a hand-back, from any copy on any account,
  is passed over the same way, naming the kind, who handed it back and when,
  while origin's default branch holds its record at the blob the tombstone
  names: it is a person's until a person changes that record, and a
  reservation the drain pushes backs off it even when the view missed it. A
  reservation refused for a reason of this checkout's own (no
  GitHub username, `origin` not answering) refuses the move, nothing opened.
  The lane's run then renews the reservation, fences its landing on it and
  tombstones it once it lands (`commands/implement.md`); when the lane is
  handed back instead, the step that hands it back pushes the tombstone,
  publishing the hand-back with its kind (`unachievable` for a lane stopped
  after its fix rounds), and the route's `wrote` says so as the drain routes
  the hand-back; a tombstone
  that step could not push (`origin` not answering) the drain pushes then. Without `origin` there
  is nowhere to reserve, and nothing is.
- **It opens a lane.** The next eligible issue in the order, once judged `no`
  over its remedy as it stands, gets the implement
  loop's issue-keyed run (the run `abcd build <iss-N>` starts): `start` names
  the run, and `lane` the issue and run id. Drive it as any run:
  `abcd implement step --run <run-id>` until it awaits an agent, then start that
  agent with the brief it names. The brief is the issue with its remedy as the
  work; the implementer's receipt names the issue in `resolves`, and the landing
  resolves it and opens one pull request.
- **It waits.** While that lane is in progress, a drain opens nothing and names
  the run again. One lane at a time.
- **It routes.** Once the lane is handed back or its pull request is open, the
  next drain records the outcome in `lanes`, routes a hand-back (below), and
  opens the next lane. A pull request the forge reports `CONFLICTING` is never
  armed: its lane stays in progress, and `abcd implement step --run <run-id>`
  refuses at exit 2 naming the conflicting files and the hand merge that
  settles it. That is a person's: report it, and do not push to the lane or
  merge it yourself. One that turns conflicting after it was armed is refused
  the same way by the run's own step; the drain's part ended at the open pull
  request.
- **It finishes merged lanes.** Every move, before it reads the window clock,
  renews the reservation of each issue lane parked at land at half its lease,
  so a pull request waiting on a person's review stays held while the drain
  keeps moving, and takes the landing's last step for each issue lane in the checkout parked at
  land whose pushed head is now on the default branch: the lane's worktree and
  branch are removed and the lane is done, during a pause included, since the
  cleanup does no new work. The lane may be this drain's, an ended drain's, or
  one `abcd build <iss-N>` started. The default branch is fetched once, and
  only while such a lane is parked; a lane whose head is not on it is left as
  it is. A lane the landing will not finish (its branch moved past what
  landed, a worktree with changes, its branch checked out in another
  checkout, which the drain names, or a pushed head git cannot read) is named
  in `next` and the move goes on.
- **It pauses.** At the end of the drain's working window it writes
  `next_eligible_at` into `.abcd/.work.local/run/drain.json` and opens nothing;
  before that time a drain opens nothing. Run it again at or after that time.
- **It ends.** At `--max <n>` lanes (`stopped: "cap"`), or when nothing eligible
  is left (`stopped: "empty"`), it reports and ends; `complete` is `true`. The
  next `abcd drain` begins a new drain.
- **It stops on a lost connection.** A drain's lane waits out a lost network or
  model-service connection as any run does (`/abcd:build` names the protocol).
  Once that run gives up, after eight hours of failed hourly probes, the next
  drain opens nothing and ends with `stopped: "outage"` and `complete: true`;
  `next` names the services down, since when, the probes made and the lanes the
  drain opened, and says how to resume: once the connection is back, close the
  outage with `abcd implement outage clear --session <id> --reason <why>`, and
  the next `abcd drain` begins a new drain. The text form reads "ended: the run
  gave up on a lost connection". Tell the user the drain stopped on an outage,
  the lanes it had opened, and `next`.

`--pace`, `--sub-agents` and `--fix-rounds` are `abcd build`'s, read when a
drain begins; `--max` is set then too. Naming another while the drain runs is
refused.

### The hand-back, by kind

A `yes` from the host judgement is routed by its kind as the table below routes
a lane's, before any lane opens. A lane that meets a decision writes `"handback": {"kind", "reason", "home"}` in
its receipt in place of `resolves`; the loop discards the lane's worktree and
branch and ends it. The drain routes it:

| `kind` | `route` | Written |
| --- | --- | --- |
| `user-visible` | `promoted`: an intent draft seeded from the issue (`capture promote`) | the draft, and the issue's `related_intents`; nothing else |
| `trust-rule` | `decision-record`: flagged, with the lane's reason as `question` | nothing; no record is minted |
| `design-finding`, `second-package` | `home`: flagged with the `home` the lane names | nothing |
| (fix rounds exhausted) | `home`: flagged, the issue staying open | nothing |

Every issue the rule hands back is in `flags` with `from: "field"`,
`route: "rule"` and the `rule` that excluded it; `flags` are re-derived at every
move and written nowhere. An issue this drain promoted itself reads as
`promoted` on every move after, and is not flagged again: it is counted once,
in `hand_backs` with `route: "promoted"` and its `draft`. So the rule
`promoted` is always a flag's `rule` field, and the route `promoted` always a
hand-back's `route` field.

### The run's payload

`state` is the drain's state file; `started` is `true` when this move began a
drain; `rule`, `loosened` and `order` are the plan's; `max` is the cap (`0`:
all); `pace` is the drain's pace; `lanes` lists every lane the drain opened
(`issue`, `run_id`, `opened_at`, `outcome`: `in-progress`, `pull-request`,
`handed-back` or `done`, and `pr`); `lane` is the one in progress; `start` is
the run this move started; `routed` are the hand-backs this move routed and
`hand_backs` every one the drain has (`issue`, `from`: `lane`, `judgement`,
`reservation` or `field`, `kind`, `route`, `draft`, `question`, `rule`, `home`, `reason`,
`wrote`); `judging` is the host judgement the drain awaits (`issue`,
`remedy_sha256`, `request`, `answer`, `requested_at`), `null` when it awaits
none; `judged` is the answer this move took and `judgements` every one the
drain has taken (`issue`, `remedy_sha256`, `answer`, `kind`, `reason`,
`applied`, `note`, `at`), `applied` being `false`, with the `note` saying why,
for an answer over an issue that left the eligible set before it was taken;
`flags` are the rule's
hand-backs; `passed` names an eligible issue this move did not take, with why;
`dispositions` is the plan; `finished` lists the merged lanes this move took
the landing's last step for (`issue`, `run_id`, `pr`, `done`, and a `note`:
the landing's line, or why it did not finish the lane when `done` is `false`);
`next_eligible_at`, `stopped` (`cap`, `empty`,
`outage` or `origin`) and `complete` say whether it paused or ended; `counts` totals the
drain as it stands: `lanes_opened`, `in_progress`, `pull_requests` (the lanes
whose landing opened one; a merge is a person's gate and is not counted),
`lanes_handed_back`, the hand-backs by route (`promotions`, `decision_records`,
`homes`), `rule_flags`, `judgements`, `passed`, and `spend` (`agent_runs`, the
sum of `implementers`, each implementer's receipt the loop verified, and
`validators`, each validator's return it read a verdict from, the unit the
budget check counts in; `unread` names a lane's run whose state could not be
read, which makes the spend a floor); `next` is the one move to make. A drain
that ends with every lane ended and none in a pull request says so in `next` as
a normal ending; one that stops with a lane still in progress does not.

Tell the user any loosened floors first, then what this move did (the
judgement asked or taken, the merged lanes finished, the lane opened, the
hand-back routed, the pause or the end), then every hand-back with its route and what it wrote, then the
counts and the spend, then `next`. Do not plan a promoted draft or write
a flagged decision: those are a person's.

## Refusals

- Without the repository's own record of the rule, the dry run and the bare
  verb refuse (exit 2, nothing written), naming how to add it: run
  `abcd ahoy install` at a terminal and accept the offer, or give an accepted
  decision record the four `drain_` fields. A record carrying the fields but
  proposed or superseded is named. Relay this; do not write the record for the
  user.
- A malformed record (a field missing or misspelt, any frontmatter key stated
  twice, an `id` its file name does not give it, or a value the field does not
  take) refuses, naming the record and the field; two accepted records carrying
  the fields refuse, naming both. A decision store or record that cannot be read
  safely (a store or record that is a symlink, wherever it points, or a record
  past the size cap) refuses. Every one of these
  exits 2 with nothing written, on the dry run and the bare verb alike.
- The run refuses (exit 2, nothing written) a checkout without the local tier,
  a negative `--max`, and a `--max` or pace other than the one a drain in
  progress began with; a drain state it cannot read as its own is refused,
  naming the file. Another drain moving in the checkout exits 3: back off and
  retry. `--dry-run` refuses `--max`, `--pace`, `--sub-agents`,
  `--fix-rounds` and `--judgement`.
- `--judgement` refuses (exit 2, nothing written) when the drain awaits no
  judgement, when the path is not the answer's path the request names, when
  the answer cannot be read as a regular file within 64 KiB, and when it is not
  exactly the request's shape: `schema_version` 1, the `issue` and
  `remedy_sha256` the request names, `answer` `yes` or `no`, a `kind` of
  `user-visible` or `trust-rule` with `yes` and none with `no`, and a `reason`
  within 4096 bytes, no other field. It refuses an answer over a remedy
  rewritten since the request; the next `abcd drain` asks again over the
  remedy as it stands. An answer over an issue that has left the eligible set
  is taken and decides nothing: the judgement never makes an issue eligible.
- Outside a checkout, or on a ledger holding one id in two status folders, it
  refuses (exit 2) as every capture verb does.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
