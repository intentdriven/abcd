---
name: drain
description: "Fix the issues needing no decision, one lane at a time, and hand the rest back: Writes its state and user-visible drafts; refuses without the rule's record."
argument-hint: "[--dry-run] [--max <n>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>] [--judgement <file>] [--take-over] [--skip <iss-N|path> --reason <why>] [--unskip <iss-N|path>]"
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

## From a plain terminal

This page always calls the drain one move at a time, and an agent never drives
it any other way. `abcd drain --drive` is the person's own form of the verb, typed
in a plain terminal with no host session; it refuses inside an agent's own session.

It runs on this account's subscription alone, on one separate login: one directory
under `~/.abcd.noindex/profiles/` that the person makes and logs the harness into
once, in their own shell. abcd never reads, stores or forwards that login. The
directory's `settings.json` is abcd's, rewritten at each start, and holds abcd's
hooks and nothing else. A credential variable the harness would bill through,
set in the environment, refuses the start, naming `itd-2611011230226453`.

Before its first move it shows the cap (eight unless `--max` says otherwise), an
estimate of the agent runs, the runner and the route, and asks once; `--yes` is
the scripted answer, and with no terminal nothing else answers. One short live
launch then checks the tools, the guard and the shell, naming any wall that does not
hold. A profile is a separate login and settings on this one account, not a boundary: a
stage's shell reads what this account can. Each stage's agent holds only the file and
shell tools its launch names, starts from an allow-list of variables, reads settings
from that directory alone, and starts only while those settings and abcd's binary
are as they were at the start. Its shell's home is emptied at each launch. A file tool
writes only in the stage's working tree and run directories. The driver's commit, merge and
reset hooks, and any filter run inside those commands, get the stage's variables, while its
network commands, its other filters and its forge calls keep the person's environment. A lane that changes the Makefile, `scripts/` or the active
hooks is handed back at its landing, and an issue that names one is passed over: that change
needs a person's own session, and the drive's last lines name each one. Where
`facilitator.mode` is `auto`, a lane's records commit takes a short review round of its own
and a lane whose reviewers passed it arms its own merge, as `abcd build`'s and any drain's
lane does. One that moves an issue's record its full round was not told of, or holds more
than plain markdown, parks before any round; one the gate leaves open says why, keeps its
reservation and is passed over for the next. A drive that stops because a run's state
changed, or could not be read, after an agent's call does not carry on from it: a later drive, step or
hand-in refuses that run until you discard its lanes (`abcd implement step --run <run-id>
--discard <lane>`) or restore its state and remove the stop the message names.

It prints one line per step and logs them under `.abcd/.work.local/logs/`. It ends
at the cap, when nothing is eligible, at the bound below, or on a stop.

**The bound.** With no flag each wait has an eight-hour limit: eight hours of runner
failures, eight hours with nothing moving, a rate limit that outlasts eight hours, or
origin unreadable through its schedule each end the run. `--until <date>` replaces
those with one bound of the person's own, either a date, read as the end of that day
on this machine's clock, or an RFC 3339 instant: the yes names it, and at that time
the run ends with the lane on its branch and its reservation released, and exits 0,
since that is what was asked. An agent in a call finishes that call first, and
`abcd drain --drive` takes the lane up later. A
long unattended run needs `--until`, and `--max` too, since the cap stays eight
lanes unless it says otherwise.

The bound is a time, not a limit on what the run spends: the yes's estimate is what
the work costs if nothing fails, and retries and probes add to it. Under a bound
nothing but the bound, the cap, an empty ledger or a stop ends the run:

- **A limit or an outage at the live check.** A limit met by the check before the
  first move is no failed check, nor is a launch that came to no verdict on any check.
  Each waits, on the schedule a limit or an outage does, with the settings checked
  before each probe, and the check runs again, so a restart during either picks the
  run up. Only a verdict, or no harness to run, is refused then. With no `--until` the
  drive refuses either way, naming which, a limit with exit 3.
- **A limit with a reset.** No agent starts, the judge included, until the time the
  harness named, however far off. A wait counts both the time slept and the wall
  clock, so a machine that sleeps through part of it wakes at the right time and a
  clock set back does not stretch it. Every four hours a short probe asks whether the
  limit still holds, and half an hour in for a reset more than eight days off: a
  rejected probe changes nothing, and the first probe answered ends the wait, which is
  read as the reset having been wrong. A reset more than 35 days off reads as a misread field
  and is treated as no reset.
- **A limit with no reset.** One short launch on the profile, asking for no tools and
  naming no brief, on the model of the launch that met the limit, asks whether the
  limit has lifted, one, five and ten minutes on and then every thirty. When it has,
  the lane's own agent starts. Only the word it asked for counts as an answer. A probe
  that gets none says nothing about the limit: after three in a row the lane's own
  agent starts and finds out, and meets the limit again if it still holds.
- **A limit that lasts.** A reset an hour or more off, or a limit that has already held
  for an hour, lets the issue go for the wait, so no peer is kept off work no one is
  doing. The lane's uncommitted work is saved aside as a stop saves it, the reservation
  is released, and the lane stays on its branch. An hour is where a renewal first falls
  due and the outage schedule turns hourly: short of it a hold costs less than the two
  commits of releasing and reserving again. With no reset named this takes a probe, so
  with no `--until` that limit keeps its hold for the launch that asks. A lane with an
  open pull request, an agent still in a call, or work a save could not take keeps what
  it holds, and asks again at the wait's next turn. When the limit lifts, the same run
  reserves the issue again, with the footprint it held, and goes on from its branch,
  which the landing's sync brings up to the default branch. If the issue has finished on
  the default branch, its record there changed since the release, a hand-back stands on
  it, or a peer holds it or a path of its footprint, the lane goes back holding nothing,
  saying which, with no hand-back published over the issue by this run, so a later drain
  can still take it once what stopped this one has passed. A record gone from the default
  branch counts as changed. When the comparison cannot be made, the drain waits. A drain's
  stop lets the issue go too, and the same comparison waits for whatever reserves it again.
  A person who resumes or takes over such a run, with or without a stop on it, reserves it
  again and goes ahead, told if the record changed, is gone or could not be compared, since
  the lane's brief reads it as it stood at the lane's base. A reservation made by hand
  during the wait stays when the lane goes back, and the lane says so. Its worktree, its branch and what it saved stay, named in the summary,
  until `abcd implement step --run <run-id> --discard <lane>`, and the run takes the next
  issue. A stop during the wait holds as any stop does, and nothing marks a lane let go
  this way as a person's: a later drive takes it up.
- **An agent that goes silent.** Silence cannot be told from a hang yet, so it is
  treated as one: the role's own deadline kills it, what it left stays in the lane's
  worktree, and the role is retried on the outage schedule. A stream that reported
  a limit before its silence, and has not reported it lifted, reads as that limit when
  the deadline falls. A launch killed with nothing spent counts against its lane only
  if a probe answers, which shows the account is alive.
- **An outage or a stuck queue.** Runner failures keep the outage schedule, at most an
  hour apart. Work held by someone else, the reservation cap, a contended lock and a
  push or fetch origin refused while reserving are looked at again at most fifteen
  minutes apart. A forge or remote answering 5xx, in gh's `HTTP 502:` form or git's
  `The requested URL returned error: 503`, waits as a lost network does, here and in a
  drain with no drive. Origin keeps its schedule and is never given up on. An outage another
  session gave up on holds the run until its probes answer, `abcd implement outage clear`
  or the bound. A
  launch that was retrying the model service when it ended, or ran out of retries with
  nothing spent, reads as an outage and says so, and with nothing spent it counts against
  nothing. Under a bound the run joins the shared run for its outages alone, so an open
  record is probed when due and its own probes never give up on it, a short probe standing
  in for the model service's canary. A record another session gave up on is looked at
  hourly and cleared when the probes of every service down answer. One act that keeps
  reading as a lost network for two hours, from the first reading with origin answering,
  is no lost network, and the lane goes back as stuck; another act, or a step that worked,
  restarts the clock. Version
  2.1.318 of the harness was seen to do this with the service unreachable: ten retries
  over about three minutes, then a failed result, with the init event before any of it.
- **Work that keeps failing.** Three failures of one role in a row that are the work's
  hand the lane back as stuck, and the run goes on to other issues. An answer the
  contract refused is the work's. With nothing spent, a result naming an API status of
  401, 403, 429 or 5xx, or a launch that ran out of retries, says nothing of the work.
  Any other failure, one that spent included, since the service may have gone after
  the spending, is put to a short probe: an answer shows the account is alive, which
  makes it the work's, and a rejection or no answer counts nothing. A judge is judged
  the same way against its issue, so one at its deadline having spent is held back while
  the account answers, and an account-wide failure does not empty the drain. Six launches
  in a row that spend and fail with no probe answered hand the lane back too, or hold the
  issue back for a judge. A launch started with the service unreachable was seen to spend
  nothing on 2.1.318, so an outage of that kind does not get there; one that spends and then
  loses the service does count.
- **A lane that cannot move.** Six steps of one lane in a row that start no agent and
  move no stage hand it back as stuck, its worktree and branch left in place, so it
  does not hold every other issue until the bound. A step that started an agent ends the
  count, and a wait on a shared outage holding an agent the lane wants, or on a claim that
  has yet to lapse from a killed run, is not counted.

Each wait says what it waits on when it begins, when its reason changes, and at least
hourly while it lasts, and an agent's call or a preflight says when it starts and every
half hour while it runs, on the screen and in the log. Through every wait the run renews
what it still holds, a lane parked at its open pull request included, read from the forge first
and only when its renewal is due. A claim that lapsed anyway, a machine asleep past its
lease, is reserved again on waking; a peer who took the issue meanwhile keeps it, and the
lane goes back as stuck, as it does when its reservation was released during the wait.
A judge's hold is the one thing left to lapse in a wait that long: the next move drops the
request and asks again, after the drain's other eligible work.

With no `--until` no probe runs: the lane's own launch finds out, as the yes said.

The first limit a run meets leaves `drive-limit-<time>.jsonl` under
`.abcd/.work.local/logs/`, and the first of each other kind of failure that may be one
leaves `drive-suspect-<kind>-<time>.jsonl`: a kill, an outage, a failed exit, an API
status. A failure that names an API status says it on screen too. Each holds the exit, the end of the event
stream with every identifier dropped and free text reduced to its length, the first
stderr line through the scanner's redactor, the reset read, what the loop decided,
each probe's answer and when the run resumed. A file stops taking the stream's detail at
64 KiB and goes on taking the answers, and the newest twenty of a kind are kept.

The first interrupt, hang-up or termination is a stop: the agent in a call finishes
that call, the lane's uncommitted work is saved aside, its reservation is released, the
lane stays on its branch, and the last lines name the run, the lane and
`abcd drain --drive`, which takes the run over and resumes it, because that terminal's
person is the one who stopped it.
A second signal kills the running agent's process group and exits at once, naming
when the reservation lapses.

The person's own `abcd implement stop` ends a drive too, at its next move, and any wait
within about a minute, a wait on a limit or an outage included. The drive then says what
holds it. A preflight queued at the load gate or running ends at the next look too, and a probe does not
start. A run that stop halted waits for `abcd implement resume --take <run-id>`, also
after the stop lifts: a new drive names it, passes over it and goes on to other work. A lane parked on its pull request that a signal halted waits the same way: the drive's end names that take-over and no resume command, and a new drive names it and passes over it. A stop and a drive's stop on a run whose state changed are two things: while both
stand the run is not settled, its work is not saved aside and its reservation lapses with
its lease. Lifting the stop leaves the other, and removing the other leaves the stop.

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
- it carries no `decision_needed:`, the marker a drain writes onto an issue it
  hands back as turning on a rule about trust or safety (below);
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
| `handback` | `decision-needed` | it carries `decision_needed:`, written by a drain's trust-rule hand-back: the decision record its question asks for is a person's to write; the reason quotes the question, and the issue stays handed back until a person writes the record and takes the field out |
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
move to move and two of them start a group at different issues as often as
chance allows. When git's
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
`decision-needed` hand-back, state its question: the decision record a person
owes. For a
`waits-on-ruling` or `deferred` hand-back, say which ruling or release it waits
on, and for a `promoted` one, which intent it goes through; a person's own
`abcd build <iss-N>` still takes a `promoted` issue no other rule would hand back, naming the intent. Do not act on the list: a hand-back is a person's decision.

## The run

Each `abcd drain` without `--dry-run` performs one move and exits 0, saying
what it did in `next`:

- **It reads the ledger from origin.** In a checkout with a remote named
  `origin`, every move but one waiting out its next probe first fetches the reservations and origin's default
  branch, under the drain's lock, and plans from the ledger that branch holds,
  never from the checkout, which may be behind: an issue another account
  finished there is not taken, and one filed there and not yet in this
  checkout is judged and opened from origin's record. `origin` names the
  branch, its commit, the commits this checkout is behind and the issues open
  here it has finished. A move that cannot read origin (the fetch fails, the
  default branch cannot be read, or none can be named: no `origin/HEAD`, and no
  `origin/main`, `master`, `trunk` or `develop`) plans nothing:
  it asks no judgement, takes no answer given (`next` says to give it again
  once origin answers), finishes no merged lane, renews no reservation,
  reserves nothing, records the failure in the drain's
  state (`origin_wait`: `since`, `kind` — `fetch`, `default-branch` or
  `origin-head` — `probes`, `next_probe_at`, and `hourly_since` once the
  probes are hourly) and says in `next` when it looks again, one minute, five,
  ten, then hourly; a move before then reads nothing and does the same. It gives up as the outage
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
  A `no` lets the lane open in that same move, unless the window has elapsed or
  the order now asks another issue first: the answer is then kept, and a later
  move opens the lane with no second question. A `yes` names its kind,
  `user-visible` or `trust-rule` (`trust-rule` when it is both), and hands the
  issue back before any lane opens, routed as a lane's hand-back of that kind
  (below), `from: "judgement"`; the drain then asks about the next eligible
  issue, in that move unless its window gate or its cap stops it first. When the remedy does not let you tell, answer `yes` and say what you
  could not tell. Do not open the lane, edit the record or fix anything while
  judging.
- **Its judge may be routed to a runner.** When the person's own
  `~/.abcd.noindex/config.json` routes the `drain-judge` role to a runner
  (`roles.drain-judge.runner: "claude"`, the runner enabled under
  `runner.claude`), the move that writes the request also starts that runner on
  a brief of its own and takes the answer it writes in that same move,
  validated as yours would be: the routed answer carries no `remedy_sha256`,
  which abcd computes itself, and no `--judgement` is given. The judge runs in
  a directory of its own, a new private one outside every working tree that
  holds its brief and its answer alone and is removed once the answer is read.
  The drain judge may read and write only its own directory: it is granted
  reading that directory and writing its answer file there, each named by its
  real absolute path, and nothing else, and it is refused reading or writing
  anywhere in the checkout. Under the runner's `dontAsk` mode anything it is
  not granted is refused rather than asked, so it can read nothing outside its
  directory and change nothing in the checkout. A
  routed judge's answer steers an unattended lane, so its `no` is a judgement
  on text a stranger may have written. A route to `opencode` is skipped with a
  diagnostic and the judge stays on the host: opencode runs under its own
  configuration's permissions, not the tools a role is granted.
  `judged.route` names the runner asked, the one that ran and its model. A
  runner that does not answer, or answers out of shape, is recorded in the
  drain's state (`judge_fallbacks`, the latest 32 kept) and the request above
  is left to you, as with no route; `next` says so. The runner is started once
  per request: `judging.routed_asked` records that it was, and a later
  `abcd drain` while the request stands starts no runner and records no
  further fallback. A judge that was never started, because its directory
  could not be made or its request could not be put to the runner, is not
  marked as asked, and the next `abcd drain` starts it. The runner runs under the drain's lock, for at
  most fifteen minutes, and stops with the drain: an interrupt or a
  termination while it judges kills its process group before the move ends,
  records no fallback, and refuses the move saying so, and the next
  `abcd drain` asks the judge again.
- **It reserves the issue.** In a checkout with a remote named `origin`, the
  drain reserves each issue on it before it asks the judgement over it, as
  `abcd reserve` does, with the path the issue's `found_at` names (a `:line`
  locator dropped) as the footprint, so every other person and computer sees it
  is taken: while it holds, no other account's drain reserves, judges or opens
  the issue. `judging.hold` records the reservation the drain took (`commit`,
  `claimed_at`, `expires_at`), and a lane a `no` opens while it holds starts on it, with no
  second push. The drain releases that reservation, and no other, whenever no
  lane will open on it: on a `yes`; on an answer over an issue that has left
  the eligible set; when the judgement is superseded (the order now asks another issue,
  nothing eligible is left, the drain stops at its cap, on an outage or giving
  up on origin, or a `no` comes once the window has elapsed); when the lane's
  start is refused or fails; and when a request lapses. It does so on every
  path the drain's process lives through, an error or a crash in the move
  included, an interrupt (Ctrl-C) or a termination signal among them: the move
  runs to its end, stopping a judge that is running. Only the first signal
  is taken this way: a second interrupt or termination stops the drain at
  once, as it would any program, which is the way out of a move waiting on a
  network act, and releases nothing then. A drain killed outright
  (`SIGKILL`) releases
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
  it, and the hand-back is published all the same. A `yes` whose hold the
  move cannot release with the hand-back on it (you renewed or reserved the
  issue again by hand, another took it over, you released it, or the drain
  took none because you held the issue live) says in `wrote` that the
  hand-back is not yet published, and why, and keeps it in
  `pending_hand_backs`; a later move whose view shows the issue released by a
  tombstone that marks no hand-back publishes it over that release, fenced on
  it and on the blob the `yes` was judged over, so your own reserve and
  release never erase it. A view that shows no branch at all, the sweep having
  deleted the release first, publishes it as the branch's first commit, which
  origin takes only while the branch is still absent: if anyone reserves the
  issue meanwhile, the hand-back stays pending for that reservation's release.
  One judged while origin's default branch could not be read names no record
  to fence on, so over a deleted branch it is dropped, and the move says so. A record changed or finished on origin's default
  branch since, one origin holds in no one file, or a tombstone the hand-back
  would take past its cap ends it with nothing published, and the move says
  in `next` and in the route's `wrote` that it was dropped, and why, never
  that it was published; a move that could not read origin's default branch
  keeps it pending. No drain takes over or renews a reservation you made over
  a standing hand-back, lapsed or not: it backs off, as it does the hand-back's
  own tombstone. The reservation is taken once
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
  move's one; one the push itself finds held is passed over the same way. When
  the holder is the user's own other copy (a hosted session that restarted
  loses its local tier, so its new copy reads its own reservations as another
  copy's, provided `ABCD_HOLDER_ID` keeps its holder id across sessions: a
  restart loses its home too, and without the variable it mints a new id, so
  its earlier reservations read as another person's, which no take-over
  moves, until they lapse), the reason says so and names `abcd drain --take-over`: run that,
  with the user's say-so, and the move reserves the issue for this copy,
  taking the reservation over, as `abcd build --take-over` does. The flag is
  given per move and never moves another person's reservation. An
  issue whose tip on `origin` is a hand-back, from any copy on any account,
  is passed over the same way, naming the kind, who handed it back and when,
  while origin's default branch holds its record at the blob the tombstone
  names: it is a person's until a person changes that record, and a
  reservation the drain pushes backs off it even when the view missed it. A
  reservation refused for a reason of this checkout's own (no
  GitHub username, `origin` not answering, `origin` a fork the repository
  does not allow) refuses the move, nothing opened; one that could not ask
  GitHub whether `origin` is a fork goes ahead, and the lane that starts on it
  says so in its run record and in `start.fork_unchecked`, as `abcd reserve`
  does, in whichever move the lane opens; a lane whose own reservation renews
  a judgement's hold that lapsed asks again. A move refused once it has done
  anything (a judgement taken, a hand-back routed, a lane finished) writes
  that first, and its refusal says whether the answer given was taken, so it
  is never given twice. A peer this checkout cannot read (a sibling worktree git will not
  answer for, one another account owns, or a ledger holding one id in two
  folders) refuses the move once, after the move's housekeeping is done and
  written and before any issue is judged or reserved (check `peers`),
  naming each peer with the repair its reason calls for: never a removal, which
  would delete the worktree's ignored files. Left alone, such a peer would
  read as holding every issue, and the drain would pass each over and end.
  The lane's run then renews the reservation, fences its landing on it, before
  its push and again before arming, and tombstones it once it lands
  (`commands/implement.md`). Each fence also reserves any path the lane's branch
  changes that the reservation does not cover, judged for overlap; another's hold
  on one stops the lane there, naming that hold. Every drain move that neither waits on origin nor stops at a given-up outage also renews the lane in
  progress at half its lease, waiting on the pace or a judgement or not, though no step of the lane ran,
  so a lane the host paused past its lease is not taken over (a lane handed
  back is renewed by no move). Once the lane's validators have passed and it
  is at its landing, the first renewal due at half its lease (a fence's renewal
  keeps the footprint) narrows its footprint to the files its
  branch changed within it, measured from where it meets `origin`'s default
  branch, so what a sync merged in is left out (a `found_at` naming `docs/`
  becomes the files under `docs/` the lane changed), and other accounts can
  take the rest. While the validators still judge it, a fix round may change
  another file, so nothing narrows. A narrowing never widens the hold; one
  that would name a file no reservation can carry is not made, and the
  renewal still is; the run record's renewal line names both footprints, or
  says `reservation kept at its full footprint` and why. A lane its landing's
  sync sends back to a fresh round reserves its whole `found_at` again before
  it goes on, judged for overlap, since a fix round may follow; when another
  reservation or the cap refuses that, the step is refused for a person,
  naming it, and the lane stays at its landing with nothing pushed. Each
  renewal names the lane's stage (`worktree`, `brief`, `implement`,
  `validate` or `land`), so another account sees where the lane stands as of
  its last renewal. A lane whose
  reservation is another person's by its push pushes nothing, opens no pull
  request, and the next drain hands it back as `stuck` at once. When the lane is
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
  agent with the brief it names. A role routed to a runner is started by
  `implement step` itself (`commands/build.md`); a claude launch whose role's
  contract grants the shell has its shell commands answered by
  `abcd guard approve`, against the hazard registry at the lane's base commit
  (`commands/guard.md`). The brief is the issue with its remedy as the
  work; the implementer's receipt names the issue in `resolves`, and the landing
  resolves it and opens one pull request. Before the landing pushes the lane's
  branch, it announces `merging <iss-N>` on your status branch and pushes only
  once that is on `origin` (`/abcd:implement` says how); a push typed by hand
  would post no announcement.
- **It waits.** While that lane is in progress, a drain opens nothing and names
  the run again. One lane at a time.
- **It hands a stuck lane back.** A lane that cannot finish does not hold the
  drain. Each refused step of the lane's run, until its pull request is armed or left
  open, is logged on the lane (`refused`
  in its state: the stage, the check, the reason and how many times in a row
  the step was refused that way), and `next` says so while the lane is in
  progress. Once the step has been refused the same way three times, or has
  kept being refused at one stage for sixty minutes, counted from its first
  refusal to its last (a quiet of more than fifteen minutes starts the time
  afresh, and the quiet after the last refusal never counts), the next drain
  hands the lane
  back as `stuck`, naming the refusal, releases its reservation with a
  tombstone marking it `stuck`, and takes up the next issue in the same move.
  A refusal that cannot change hands the lane back at once: its issue finished
  at the lane's base or on `origin`'s default branch, its reservation now
  another person's, or its sync's merge conflicting in a file not declared
  generated. A wait is not a refusal and is never counted: the landing's wait
  for its preflight receipt, the wait through an outage, a land fence that
  cannot read `origin`, and a release being cut that holds merges.
  The bounds are `pace.stuck_refusals` and `pace.stuck_minutes` in
  `.abcd/config.json` or `~/.abcd.noindex/config.json`, read at every move that reads a lane in progress. A
  stuck lane keeps its worktree and branch:
  `abcd implement step --run <run-id> --discard <lane-id>` removes them, closes
  its pull request if it opened one, and ends the run. The run ended this way
  still reads as handed back, not as one whose landing settles the issue, so
  once its record changes on `origin` the drain takes the issue up again, as
  it does any issue it handed back.
- **It routes.** Once the lane is handed back or its pull request is open, the
  next drain records the outcome in `lanes`, routes a hand-back (below), and
  opens the next lane. Before its records and its push, the lane's sync
  fetches `origin`'s default branch and merges it whenever it has moved, a fresh
  round judging the merge (`commands/implement.md`): a conflict confined to
  files `origin` declares `linguist-generated` goes back to an implementer to
  regenerate them, and any other conflict refuses the step naming the files,
  nothing pushed, so the next drain hands the lane back as `stuck` at once. A
  pull request that starts to conflict at its arming, or while it waits
  parked, on declared generated files alone or in no file when this checkout
  merges enters the sync again (the
  drain's next move takes that step for a parked lane, withdrawing an armed
  merge in a step of its own, reads it as `syncing` and drives the lane
  again; a step that cannot enter the sync is named beside the lane's
  reading); one whose head already holds the default branch waits at exit 3
  for the forge's stale verdict; one that conflicts on any other file is not
  armed, or stays as it is when it is armed already, and `abcd implement step --run <run-id>` refuses at exit 2 naming the
  conflicting files and the hand merge that settles it. That is a person's:
  report it, and do not push to the lane or merge it yourself.
- **It finishes merged lanes.** Every move that neither waits on origin nor stops at a given-up outage,
  before it reads the window clock,
  renews the reservation of each issue lane parked at land at half its lease,
  so a pull request waiting on a person's review stays held while the drain
  keeps moving, and takes the landing's last step for each issue lane in the checkout parked at
  land whose pushed head is now on the default branch: the lane's worktree and
  branch are removed and the lane is done, during a pause included, since the
  cleanup does no new work. The lane may be this drain's, an ended drain's, or
  one `abcd build <iss-N>` started. The default branch is fetched once, and
  only while such a lane is parked. A lane the landing will not finish (its branch moved past what
  landed, a worktree with changes, its branch checked out in another
  checkout, which the drain names, or a pushed head git cannot read) is named
  in `next` and the move goes on.
- **It reads parked lanes from their pull requests.** A move that finds a
  lane parked at land lists your latest 100 pull requests once
  (`gh pr list --state all --author @me`), and reads each parked lane whose
  head is not on the default branch from its own: a pull request closed
  without merging hands its lane back as `closed`, releasing its reservation
  with a tombstone marking it `closed`, and the drain routes it home to a
  person (`abcd implement step --run <run-id> --discard <lane-id>` removes the
  lane's worktree and branch); a conflicting one is no longer renewed, and
  `next` names it; an open one is renewed and named as waiting. An open one
  the lane armed that the listing shows disarmed, or armed and `CLEAN` (the
  listing names no queue entry), is watched in the merge queue by the lane's
  own landing step, as `abcd implement step` watches it
  (`commands/implement.md`): one left outside the queue at `CLEAN` is queued,
  and one that left the queue disarmed, as a failed merge-group check leaves
  it, is read as `dropped`, `next` naming the forge's reason and the failed
  check; it is still renewed, never queued or armed again by the drain, and
  waits for a person to re-arm it. One the forge refused to queue whose
  auto-merge the step toggled, and whose new arming the forge then refused,
  is read as `dropped` too, `next` saying the loop left it disarmed and
  naming the `gh pr merge` command that arms it again. One that merged
  after the move fetched the default branch is finished by that step and
  named among the finished lanes. One merged by
  a squash or rebase, its head not on the default branch, is named and no
  longer renewed. A pull request the listing does not show, or a listing that
  fails, keeps its lane held as before, and says so.
- **It pauses.** At the end of the drain's working window it writes
  `next_eligible_at` into `.abcd/.work.local/run/drain.json` and opens nothing;
  before that time a drain opens nothing. Run it again at or after that time.
- **It waits at the cap and on held work.** When the user holds their cap of
  live reservations (ten unless `reservation.max_live` in their machine's
  `~/.abcd.noindex/config.json` sets another, from 1 to 100), counted in the move's view
  before any reservation or judgement, or when every eligible issue left is
  held by another, the move neither refuses nor ends: it reserves and asks
  nothing, writes `waiting_until` fifteen minutes on (`waiting` says `cap` or
  `held`), and names what it waits on: at the cap, the reservations and the
  pull requests waiting to merge, with the advice to wait for one to merge
  rather than release one; on held work, each hold. It keeps every judgement
  it has taken, and the pace window is left as it was. Run it again at or
  after `waiting_until`; a move before it still looks, and takes any issue
  freed since.
- **It ends.** At `--max <n>` lanes (`stopped: "cap"`), or when nothing eligible
  is left, held or not (`stopped: "empty"`), it reports and ends; `complete` is
  `true`. The next `abcd drain` begins a new drain.
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
drain begins, and every lane the drain opens runs on them, not on what the
configuration resolves when it opens. Given while the drain runs, they are
taken: each number named replaces the one the drain runs on, `pace` shows the
new pace, `next` says the drain took it, the next lane opens on it, and the
lane in progress takes it from its next stage. A move that is refused takes
none of it, in the drain or in a lane, and says so; give it again. `--max` is set when a drain
begins; naming another while it runs is refused.

An issue the drain handed back is passed over only while its record stands on
`origin`'s default branch: once a person changes that record there, the drain
judges it afresh and may open a new lane for it, though the handed-back run's
directory remains (`hand_backs[].blob` is the record's blob as it was routed).
That new run is then the issue's: `abcd build <iss-N>` resumes it, and
`abcd implement step` or a record read with no `--run` addresses it, the
handed-back run no longer counting as one in progress.
Every other run of an issue in this checkout passes it over, in progress or
complete, on `origin` or not: a complete run landed its pull request, which may
still be open, and the drain never opens a second. Without a remote named
`origin`, a handed-back issue stays passed over, and so does an issue whose
run in this checkout was handed back.

The move that begins a drain says once, in one line on stderr (stdout's JSON is
untouched), that approval prompts are refused while it runs: the host's
approval prompt for a shell command in this checkout or in a lane the host runs is
refused rather than shown, and the agent is told to rewrite the command into
plain steps abcd can check, or skip the step and say why in its report, as
`/abcd:build` describes. The refusal never turns into an approval; a lane
started through the claude runner runs with the host's hooks off, and the
guard answers its prompts instead (`abcd guard approve`).

Refusal stops when the drain stops, or after two hours with no activity in the
drain. A lane handed back does not stop it, since the drain routes the
hand-back and moves on to the next issue unattended. No verb ends a drain
early, so a person who interrupts a drain and carries on by
hand sees the prompts again two hours after it last moved.

A drive's moves print no such line: it starts from a plain terminal. Otherwise:

**A session whose setting reads as off a profile is warned while the away mode
holds.** While `abcd mode autonomous` holds, or its record cannot be read, every
move of a drain, not only the first, prints, after the move, one line on stderr
opening `WARNING:` when the setting `/abcd:mode` describes reads as off a profile,
and says why when the record was the doubt: a later move may be another session's.
Pass the line on to the person, as it asks. Nothing is refused and the exit code
does not change. In the managed mode and on `--dry-run` it prints nothing.
This reads one setting and proves no wall, and the setting's value changes nothing
the session holds.

### Holding an issue back for now

To keep the drain off one issue, or off every issue whose `found_at` lies in
one path (a package a fix is in flight in), hold it back with a reason:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" drain --skip internal/core/capture --reason "a peer's fix is in flight there" --json
"${CLAUDE_PLUGIN_ROOT}/abcd" drain --unskip internal/core/capture --json
```

The move adds the entry to the drain's skip list (beginning a drain when none
is in progress) and then moves as usual; every move that takes up the eligible
issues passes a held-back issue over before anything is pushed or asked, in `passed` with check `skipped` and
the reason. `--skip` may be repeated, all with the one `--reason`; a target
already listed takes the new reason; `--unskip` takes an entry off. The list
lives in the drain's state, writes nothing onto the issue, and clears when the
drain ends; a drain left with only held-back issues ends, clearing it.

### The hand-back, by kind

A `yes` from the host judgement is routed by its kind as the table below routes
a lane's, before any lane opens. A lane that meets a decision writes `"handback": {"kind", "reason", "home"}` in
its receipt in place of `resolves`; the loop discards the lane's worktree and
branch and ends it. The drain routes it:

| `kind` | `route` | Written |
| --- | --- | --- |
| `user-visible` | `promoted`: an intent draft seeded from the issue (`capture promote`) | the draft, and the issue's `related_intents`; nothing else |
| `trust-rule` | `decision-record`: flagged, with the lane's reason as `question` | `decision_needed:` on the open issue, holding the question; nothing else; no record is minted |
| `design-finding`, `second-package` | `home`: flagged with the `home` the lane names | nothing |
| (fix rounds exhausted) | `home`: flagged, the issue staying open | nothing |
| `stuck` (the lane kept refusing) | `home`: flagged, the issue staying open, naming the discard of the lane's run | nothing |
| `closed` (its pull request closed unmerged) | `home`: flagged, the issue staying open, naming the discard of the lane's run | nothing |

Every issue the rule hands back is in `flags` with `from: "field"`,
`route: "rule"` and the `rule` that excluded it; `flags` are re-derived at every
move that plans and written nowhere. An issue this drain promoted itself reads as
`promoted` at every plan after whose ledger carries the promotion, and is not flagged again: it is counted once,
in `hand_backs` with `route: "promoted"` and its `draft`. So the rule
`promoted` is always a flag's `rule` field, and the route `promoted` always a
hand-back's `route` field.

The decision-record marker is the one write a trust-rule hand-back makes, the
judgement's as a lane's: `decision_needed: <question>` added to the open
issue's frontmatter, every other line of the record left as it was, so a
person finds the decision it owes without this summary. It is written in this
checkout, uncommitted, as a promotion is; a move planned from `origin` writes
it only when this checkout holds the record as origin's default branch does,
and otherwise the route's `wrote` says why it did not. The marked issue reads
as `decision-needed` at every plan after whose ledger carries the marker, and, as with a promotion, this
drain counts it once, in `hand_backs` with `route: "decision-record"`; a later
drain, and the dry run, once the ledger they read carries it, flag it by the rule `decision-needed` with the
question as its reason. Once the decision record is written, take the
`decision_needed:` line out of the issue; the next drain judges it afresh.

### The run's payload

`state` is the drain's state file; `started` is `true` when this move began a
drain; `rule`, `loosened` and `order` are the plan's; `max` is the cap (`0`:
all); `pace` is the drain's pace; `lanes` lists every lane the drain opened
(`issue`, `run_id`, `opened_at`, `outcome`: `in-progress`, `pull-request`,
`handed-back`, `done`, `halted`, a lane a person's stop halted that waits for its
take-over, or `taken-over`, the same lane once that happened and its run is the
person's to drive, and `pr`); `lane` is the one in progress; `start` is
the run this move started; `routed` are the hand-backs this move routed and
`hand_backs` every one the drain has (`issue`, `from`: `lane`, `judgement`,
`reservation` or `field`, `kind`, `route`, `draft`, `question`, `rule`, `home`, `reason`,
`wrote`, `blob`); `judging` is the host judgement the drain awaits (`issue`,
`remedy_sha256`, `request`, `answer`, `requested_at`), `null` when it awaits
none; `judged` is the answer this move took and `judgements` every one the
drain has taken (`issue`, `remedy_sha256`, `answer`, `kind`, `reason`,
`applied`, `note`, `at`), `applied` being `false`, with the `note` saying why,
for an answer over an issue that left the eligible set before it was taken,
`hold` the reservation it was asked on, and `route` the runner that answered
(`asked`, `ran`, `model`) when the judge is routed, absent for yours;
`skips` is the drain's skip list (`target`, an issue id or a path; `reason`;
`at`), empty once the drain has ended, which the text form lists as
`held back:` lines;
`pending_hand_backs` are the hand-backs routed and not yet published on
`origin` (`issue`, `kind`, `blob`, `why`, `at`), which the text form lists as
`hand-back pending:` lines;
`flags` are the rule's
hand-backs; `passed` names an eligible issue this move did not take, with why;
`dispositions` is the plan; `finished` lists the merged lanes this move took
the landing's last step for (`issue`, `run_id`, `pr`, `done`, and a `note`:
the landing's line, or why it did not finish the lane when `done` is `false`);
`parked` reads every other lane parked at land from its pull request
(`issue`, `run_id`, `pr`, `state`: `waiting`, `conflicting`, `syncing`,
`dropped`, `closed`, `merged` or `unread`, and a `note`);
`waiting_until` and `waiting` (`cap` or `held`) say when a waiting drain looks
again and on what; `next_eligible_at`, `stopped` (`cap`, `empty`,
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
  a negative `--max`, and a `--max` other than the one a drain in progress
  began with; a drain state it cannot read as its own is refused,
  naming the file. Another drain moving in the checkout exits 3: back off and
  retry. `--dry-run` refuses `--max`, `--pace`, `--sub-agents`,
  `--fix-rounds`, `--judgement`, `--take-over`, `--skip`, `--reason` and
  `--unskip`.
- The run refuses (exit 2, nothing written) a `--skip` or `--unskip` target
  that is neither an issue id nor a clean repository-relative path, a `--skip`
  without `--reason` (or a blank one, or one past 512 bytes), a `--reason`
  without `--skip`, and a skip list that would pass 64 entries.
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
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

**User input:** $ARGUMENTS
