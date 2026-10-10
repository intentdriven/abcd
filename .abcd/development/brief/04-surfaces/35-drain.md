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
[adr-2609291342092738](../../decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when-its-fields-say-it.md),
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
`blocked_by`, a category the rule takes, a severity it takes, and a `remedy:`
other than `none (filed automatically)`, the value an automatic filer writes
when it has no fix. Every open issue receives exactly one disposition, from the
first rule that excludes it: skipped when blocked; handed back for `security`
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
record by its fields as it judges any other: a failed audit's record is a
`major` `bug`, which the baseline hands back on severity, and an undecided
one's is a `minor` `inconsistency`, which the baseline takes.

Two hand-backs hold whatever the repository's record says, because each marks a
decision a person still owes. A remedy that opens "Waits on" as words, followed
by a blank, a colon or nothing (the shape a remedy takes when its fix waits on an
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
before `critical`, as far as the record takes them; then oldest first. The
record's lists are sets: the order is abcd's. The dry run states the order and
lists the eligible issues first in it, then every other open issue by id.

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
and written nowhere, decision 8), then does the first of these that applies:

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
5. It reads what that lane has come to: its pull request opened (armed, or left
   open where the repository has no merge queue), the run complete, or the lane
   handed back, which it routes (below).
6. When the drain has opened as many lanes as its cap, it ends and says so.
7. For the next eligible issue in the drain order that the drain has not taken
   or handed back and this checkout has no run for, it asks the host judgement
   over the issue's remedy (below) and opens nothing, unless the host has
   already answered no over the remedy as it stands. With that no, it starts
   the implement loop for the issue and names the run. An issue the loop's own
   checks refuse (a peer holds it) is passed over, named with the check. When
   none is left, the drain ends and says so.

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
remedy and its record's body, each quoted inside a fence longer than any
backtick run in it, and the answer's exact shape; it records in its state the
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

Every issue the rule itself hands back (by category, severity, security, a
ruling or a deferral) is flagged in the summary naming the rule. Every route is
in the summary, in text and in the machine-readable payload, with the record
change it made, so nothing is dropped silently.

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
  spc-2609212015054359, which stays open for the summary's remaining counts,
  the decision-record marker and the issues its plan names.
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

<!-- surface-appendix:end -->
