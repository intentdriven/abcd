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

The rule is a recorded decision,
[adr-2609291342092738](../../decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when-its-fields-say-it.md),
and invariant 19 in
[`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md). It reads
the record's fields and nothing else: nothing open in `blocked_by`, a category
in the fixable set, severity `nitpick` or `minor`, and a `remedy:` other than
`none (filed automatically)`, the value an automatic filer writes when it has
no fix. Every open
issue receives exactly one disposition, from the first rule that excludes it:
skipped when blocked, handed back for `security`, for a category outside the
fixable set or for a severity above `minor`, ineligible without a remedy or
with the automatic filers' value until a person writes one, and
unreadable when the ledger reader refuses the record. A record written before
the field existed reads its `suggested_fix:` as its remedy.

The fields are the rule because a model's judgement of its own ambiguity is
unreliable, and the failure runs one way: a machine that decides a thing needs
no decision, and then makes one. The host judgement over an eligible remedy is
therefore allowed only to hand an issue back; it does not run in the dry run,
and the dry run says so beside every eligible issue.

## The order

Eligible issues are taken by category, `tech-debt`, `documentation`,
`inconsistency`, `drift`, `bug`, `ux` (clean-ups and text first, on the evidence
that they merge most often), then `nitpick` before `minor`, then oldest first.
The dry run states the rule and lists the eligible issues first in that order,
then every other open issue by id.

## What it refuses

The bare verb refuses to start, exit 2, with nothing read or written: the lane
it would hand each issue to does not exist. The start check also refuses naming
the decision record it needs when the rule has none; the binary names the rule's
record, and a test holds that name to an accepted record here. A checkout that
cannot be resolved, or a ledger holding one id in two status folders, is refused
as every capture verb refuses it.

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
