---
name: drain
description: "Fix the issues needing no decision, one lane at a time, and hand the rest back: Writes its state and user-visible drafts; refuses without the rule's record."
argument-hint: "[--dry-run] [--max <n>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>]"
block: agents
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
| `handback` | `security` | category `security`, a person's unless the rule sets `drain_security: take` |
| `handback` | `category` | a category the rule does not take |
| `handback` | `severity` | a severity the rule does not take (`major` and `critical` under the baseline) |
| `handback` | `waits-on-ruling` | its remedy opens "Waits on": the fix waits on a ruling a person has not given |
| `handback` | `deferred` | its `deferred_after` names the current anchor tag: a person carried it past this release; or the checkout holds no release tag (a shallow clone fetches none), so whether any deferral is live is unknown and the reason names `git fetch --tags`; or it names a release tag newer than the checkout's own, which the checkout lacks, so the reason says `anchor stale` and names that tag and `git fetch --tags`; or it is not a release tag at all |
| `ineligible` | `remedy` | no remedy (a record filed before the remedy was required), or `none (filed automatically)` from an automatic filer; ineligible until a person writes one with `abcd capture remedy`, which the reason names |
| `unreadable` | `unreadable` | the ledger reader refuses the record; the reason names why |
| `eligible` | `fields` | every field rule passes |

The host judgement over an eligible remedy (a user-visible or trust-boundary
change hands it back) is not built: the run opens a lane for every eligible
issue, and only the lane itself can hand its issue back.

## The order

Eligible issues come first, in the order a drain takes them: by category
`tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux`, then
`security` when the rule takes it; then by severity, `nitpick` first; then
oldest first. The payload's `order` states the rule. Every other open issue
follows, by id.

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

Tell the user any loosened floors, then the counts, then the eligible issues in
order, then the others grouped by outcome with their reasons. For an
`ineligible` issue, say that a person writing a remedy with
`abcd capture remedy <iss-N> "<fix>"` is what makes it a candidate, and name the
ones an automatic filer wrote apart, since their reason says so. For a
`waits-on-ruling` or `deferred` hand-back, say which ruling or release it waits
on. Do not act on the list: a hand-back is a person's decision.

## The run

Each `abcd drain` without `--dry-run` performs one move and exits 0, saying
what it did in `next`:

- **It opens a lane.** The next eligible issue in the order gets the implement
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
  opens the next lane.
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

A lane that meets a decision writes `"handback": {"kind", "reason", "home"}` in
its receipt in place of `resolves`; the loop discards the lane's worktree and
branch and ends it. The drain routes it:

| `kind` | `route` | Written |
| --- | --- | --- |
| `user-visible` | `promoted`: an intent draft seeded from the issue (`capture promote`) | the draft, and the issue's `related_intents`; nothing else |
| `trust-rule` | `decision-record`: flagged, with the lane's reason as `question` | nothing; no record is minted |
| `design-finding`, `second-package` | `home`: flagged with the `home` the lane names | nothing |
| (fix rounds exhausted) | `home`: flagged, the issue staying open | nothing |

Every issue the rule hands back is in `flags` with `route: "rule"` and the
`rule` that excluded it; `flags` are re-derived at every move and written
nowhere.

### The run's payload

`state` is the drain's state file; `started` is `true` when this move began a
drain; `rule`, `loosened` and `order` are the plan's; `max` is the cap (`0`:
all); `pace` is the drain's pace; `lanes` lists every lane the drain opened
(`issue`, `run_id`, `opened_at`, `outcome`: `in-progress`, `pull-request`,
`handed-back` or `done`, and `pr`); `lane` is the one in progress; `start` is
the run this move started; `routed` are the hand-backs this move routed and
`hand_backs` every one the drain has (`issue`, `from`, `kind`, `route`, `draft`,
`question`, `rule`, `home`, `reason`, `wrote`); `flags` are the rule's
hand-backs; `passed` names an eligible issue this move did not take, with why;
`dispositions` is the plan; `next_eligible_at`, `stopped` (`cap`, `empty` or
`outage`) and `complete` say whether it paused or ended; `next` is the one move
to make.

Tell the user any loosened floors first, then what this move did (the lane
opened, the hand-back routed, the pause or the end), then every hand-back with
its route and what it wrote, then `next`. Do not plan a promoted draft or write
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
  retry. `--dry-run` refuses `--max`, `--pace`, `--sub-agents` and
  `--fix-rounds`.
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
