# `/abcd:drain` — Sort the Open Ledger by What Needs No Decision

`/abcd:drain` is the verb a person types to have the open issue ledger worked
unattended: the issues that need no decision are fixed, and the rest are handed
back to the place a person decides them (itd-82, spc-2609212015054359). This
chapter describes the part that ships, the field-only slice: the rule that
decides which issues a machine may take alone, the order it takes them in, and
a dry run that shows every open issue's disposition and writes nothing. The run
itself, which hands each eligible issue to the implement loop keyed by the
issue, is not built, so the bare verb refuses to start and says so.

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
deferral, the dry run naming the missing tags and `git fetch --tags`.

The fields are the rule because a model's judgement of its own ambiguity is
unreliable, and the failure runs one way: a machine that decides a thing needs
no decision, and then makes one. The host judgement over an eligible remedy is
therefore allowed only to hand an issue back; it does not run in the dry run,
and the dry run says so beside every eligible issue.

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
warning on stderr naming each loosened floor; and the start's refusal names
them.

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
  trust-boundary reader, so a store that is a symlink leaving the checkout, a
  record that is a symlink at all, and a record past the ledger's size cap are
  refused rather than followed or read whole.
- The two person-owed hand-backs, a remedy waiting on a ruling and a live
  deferral, hold whatever the record says.

## What it refuses

The dry run and the bare verb both refuse, exit 2 with nothing written, when
the repository holds no accepted record of the rule, naming how to add one
(the setup verb's offer, or the four fields on an accepted record); when a record
names the fields but is proposed or superseded, the refusal names it. They
refuse a malformed record, naming the record and the field, two accepted
records, naming both, and a store or record that cannot be read safely. With the rule, the bare verb still refuses to start: the
lane it would hand each issue to does not exist, and the refusal names the
rule's record and every floor it loosens. A checkout that cannot be resolved,
or a ledger holding one id in two status folders, is refused as every capture
verb refuses it.

## Where this sits

- The intent and its decisions: itd-82; the design record:
  spc-2609212015054359, which stays open for the run, the judgement, the
  hand-back writes, the pace and the caps.
- The lane it will hand issues to: itd-2609201916151817 decision 10, and
  [`34-build.md`](34-build.md).
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

<!-- surface-appendix:end -->
