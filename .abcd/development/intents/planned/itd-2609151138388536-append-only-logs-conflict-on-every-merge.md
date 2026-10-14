---
id: itd-2609151138388536
slug: append-only-logs-conflict-on-every-merge
spec_id: spc-2610110931381316
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
related_issues: [iss-2609100507439414]
origin: extracted-from-record
production_mode: hand-written
related_adrs: [adr-2610110925465593, adr-2609151138420062]
---

# Every Decision Is Its Own Record, and the Log Stops Conflicting

Typed links: `related_adrs` [adr-2610110925465593](../../decisions/adrs/2610110925465593-decisions-are-one-flat-file-each-and-the.md) (the shape rule this intent enacts) and [adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted.md) (the earlier rule it supersedes); `refines` [adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and.md) (the timestamp-numeric id seam, which this adds a sixth family to and changes in no way); `related_issues` [iss-2609100507439414](../../../work/issues/open/iss-2609100507439414-append-only-logs-conflict-on-every-merge.md) (the negative finding: the append-only log conflicted on the first two of 27 branch merges). Prose cross-references, not typed links, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed.md)): [itd-2609150819432059](../shipped/itd-2609150819432059-abcd-launch-cannot-set-up-the-release.md), the companion that closes the OTHER conflict class — this intent refines nothing of its, and it refines nothing of this intent's, but the two together close both classes and neither closes both alone; and [iss-2609100508570803](../../../work/issues/resolved/iss-2609100508570803-the-record-verbs-worked-from-worktrees.md), the positive finding from the same run that makes this a shape question — five one-file-per-entry record families, zero conflicts, same 27 merges.

## Press Release

> **abcd's decision log becomes a folder of minted records, and the merge
> conflict it produced on every parallel branch goes away.** A decision is no
> longer a line appended to the bottom of one shared file. It is a record with
> an id, allocated through the same seam that mints captures, intents, specs and
> ADRs, written to its own file in one flat folder, `.abcd/work/decisions/`.
> A command and the site list every decision in date order. The old single file,
> `.abcd/work/DECISIONS.md`, is retired, and every link and citation that named
> it now points at the decision's own file. Two branches that each record a
> decision now merge the way two branches that each capture an issue already do:
> git sees two new files and has nothing to reconcile. A repository abcd manages
> records its new decisions the same way; abcd never touches its existing log,
> which its owner can convert with the same one-off script.
>
> "I landed twenty-seven branches in a day and the decision log fought me on the
> first two merges — I ended up hand-writing a merge attribute into
> `.gitattributes` at two in the morning to make it stop," said Maya,
> autonomous-development practitioner. "The issue ledger never once did that, and
> nobody had to think about why. Now the decisions behave like the issues do."

## Why This Matters

Graduated from [iss-2609100507439414](../../../work/issues/open/iss-2609100507439414-append-only-logs-conflict-on-every-merge.md):
a managed repository's shared append-only files conflicted on nearly every merge
across an autonomous run that landed 27 worker branches through one integration
branch in a single day. `.abcd/work/DECISIONS.md` conflicted on the first two
merges and would have conflicted on every later one; the session stopped it by
hand, by adding a `merge=union` attribute for that path to `.gitattributes`.

The positive finding from the same run,
[iss-2609100508570803](../../../work/issues/resolved/iss-2609100508570803-the-record-verbs-worked-from-worktrees.md),
is what makes this a shape question rather than a plumbing question. In the same
27 merges — 33 issue resolutions, 5 intents shipped, 3 ADRs superseded, 1 ideate
verdict — the record store did not conflict once. Every one of those five
families is one file per entry. The two files that conflicted are the two that
are single, append-to-the-bottom files. The conflict is a property of the file
shape, not of the content or of the branching.

The log itself has said so since it was written. `.abcd/work/DECISIONS.md`'s own
header carries the instruction: "Graduate this file to per-file
`decisions/<date>--<slug>.md` if size or parallel-agent merge contention bites."
It has now bitten, in a measured run, at 330 entries and 2,489 lines.

The remedy applied in the field — the union merge attribute — works, and it is
the reason the log stopped conflicting for the rest of that run. It is still a
patch on the shape rather than a fix of it: it needs a `.gitattributes` entry
written into a repository abcd adopts, it makes the ordering of two concurrent
appends non-deterministic, and it is safe only for entries that are independent
and dated, which is a property nothing checks. A folder needs none of that.

This intent closes one of the run's two conflict classes. The other is the
changelog, whose `[Unreleased]` section has the same shape and where union is
**not** safe — it duplicates the `###` headings. abcd answered that half for
itself by deriving the changelog from records at the cut and refusing a non-empty
`## [Unreleased]`; what a managed repository lacks is the ability to use that
flow, which is
[itd-2609150819432059](../shipped/itd-2609150819432059-abcd-launch-cannot-set-up-the-release.md).
This intent `refines` nothing of itd-2609150819432059's, and itd-2609150819432059
`refines` nothing of this one's: they are disjoint. Together they close both
conflict classes, and neither closes both alone. The typed link is a companion
link, recorded so that a reader who finds one half does not conclude the other
half was overlooked.

The shape rule this delivers is
[adr-2610110925465593](../../decisions/adrs/2610110925465593-decisions-are-one-flat-file-each-and-the.md),
which supersedes
[adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted.md).
The id seam it mints through is
[adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and.md),
which is already the allocator for captures, intents, specs and — since the
ruling of 2026-09-01 — ADRs; a sixth family costs the seam nothing, which is the
point of having a seam.

## Mechanism

We expect this to remove the conflict class rather than reduce it **because a
merge conflict requires two sides to have edited overlapping regions of the same
file, and a folder of one-file-per-decision gives two concurrent decisions no
shared region at all.** The prediction is falsifiable in the shape of the
artefact, not in a statistic: if the assembled index is a committed file that
every decision also writes to, the conflict simply moves from `DECISIONS.md` to
the index, and the class is not closed — it is renamed. The index must therefore
be derived at read time or regenerated from the folder, never appended to by the
minting verb, and an acceptance criterion below is written to catch exactly that
failure.

The secondary claim is that the readability cost is recoverable. A folder is
harder to read end to end than a file, and **we expect the listing command and
the site page to recover that**, because each renders the whole history in date
order from the folder, and every citation that named the old file now names the
decision's own file. If a reader has to open hundreds of files to answer a
question the single file answered, the listing has failed and the claim is
wrong.

## Scope Conditions

- **Repositories using git.** The claim is about git's merge behaviour on file <!-- cond: cond-2610110931385884 -->
  boundaries; nothing here is asserted about another version-control system.
- **abcd itself and every repository abcd manages.** New decisions take the <!-- cond: cond-2610110931387631 -->
  shape in both. abcd never rewrites a managed repository's existing log; until
  its owner converts it by hand, it stays as frozen history, so the claim covers
  the decisions recorded after the shape is adopted.
- **Parallel branch work.** The benefit is proportional to concurrency. At one <!-- cond: cond-2610110931381195 -->
  branch at a time the single file never conflicts and the folder buys nothing
  but consistency with the other five families.
- **Decisions that are independent and dated.** This is the same condition the <!-- cond: cond-2610110931383463 -->
  union attribute silently assumed. A decision family where one entry amends
  another in place is outside this claim — an amendment is a new record, on the
  standing rule that a durable record is superseded rather than edited.

## Acceptance Criteria

> _BDD (the [itd-1](../disciplines/itd-1-acceptance-gates.md) discipline)._

- **Given** a repository on the new shape, **when** a decision is recorded,
  **then** it is written as one file under the decisions folder carrying an id
  allocated through the adr-45 seam, and no other committed file is appended to
  by that act.
- **Given** a decisions folder holding entries, **when** a person runs the
  listing command or opens the site's decisions page, **then** every entry is
  listed in date order with its id, derived from the folder, and no committed
  file is written by recording a decision.
- **Given** the shape has landed in abcd, **when** the repository is searched,
  **then** `.abcd/work/DECISIONS.md` does not exist, no committed file names it
  as a live path, and every citation that named a dated entry of it points at
  that entry's own file. A citation the migration could not map with certainty
  is listed for review in the change, never guessed.
- **Given** two branches cut from the same base that each record one decision,
  **when** both are merged into the integration branch, **then** both merges
  complete with no conflict and with both decision files present, and this holds
  with no `merge=union` attribute and no merge driver configured for the path.
- **Given** a repository abcd manages that already has a `DECISIONS.md`,
  **when** abcd's install runs, **then** the existing file is left byte for byte
  as it was, new decisions are recorded in the decisions folder, and no merge
  attribute is scaffolded into its `.gitattributes`.
- **Given** a repository adopting abcd with no decision log, **when** adoption
  runs, **then** it gets the decisions folder and no single log file.
- **Given** the new shape is in force, **when** the repository is inspected,
  **then** the decisions-append gate (`record-lint decisions-append`, rules
  DA001–DA004, in `internal/core/lint/decisionsappend.go`) is gone, no CI job invokes it, `make preflight` does not run it,
  and nothing in the record or the conventions names it as a live gate — a folder
  of minted files has no mid-file position to police, so the gate is retired by
  the shape rather than disabled.
- **Given** abcd's existing log, **when** the migration has run,
  **then** every entry is present as its own file with its original date and text
  preserved verbatim, the index renders them in date order, and the
  count matches — a migration that silently drops or reorders history fails this
  criterion.

## Open Questions

None open: the product thinker settled them on 2026-10-11 (Decisions, below).

## Decisions

- 2026-10-11, the product thinker, in the planning interview: `.abcd/work/DECISIONS.md` is retired outright, not kept as a symlink or a signpost page, and every link to it across the repository is rewritten, since a single decisions file is not a common standard (MADR and adr-tools keep one file per decision). This supersedes the earlier ADR's symlink (adr-2610110925465593).
- 2026-10-11, the product thinker: the folder is flat, `dec-<id>-<slug>.md`, with no year, month or day subfolders; the id is the timestamp, so a listing sorts by date.
- 2026-10-11, the product thinker: a repository abcd manages but did not create keeps its existing `DECISIONS.md` as frozen history, never rewritten; new decisions go to the folder.
- 2026-10-11, the product thinker, in the criteria walk: all eight acceptance criteria stand as written (four walked as stories, four accepted as a list).
- 2026-10-11, the product thinker: the migration of abcd's own log (splitting its entries into the folder and rewriting every link and citation) is a one-off script run once in this repository, not a verb or any code in abcd. It is needed once and only here; a managed repository is never migrated, since its log stays as frozen history.
- 2026-10-11, the technical facilitator, with one answer defensible on the record: the folder is `.abcd/work/decisions/` (the home the press release named) and its records are `dec-`, a prefix no family uses; the union attribute for `.abcd/work/DECISIONS.md` goes in the same change; the changelog's attribute stays out of scope as a separate conflict class.

- 2026-10-14, the product thinker, in a second planning interview that reconciled this record with later rulings: the press release's last sentence now reads "abcd never touches its existing log, which its owner can convert with the same one-off script"; a person may run the one-off script by hand on a repository they own, and abcd never runs it there; the migration of abcd's own log lands in the same change as the folder; the migrated entries take date order (the 48 written out of date order move to their dates), so the migration criterion reads "date order"; the site page is the "Decision log". The mechanism, the scope conditions and the eight criteria were walked again and stand.
- 2026-10-14, the technical facilitator: the prefix is `dcn`, not `dec`, so a decision record reads apart from the lifeboat's `dec-L<line>` keys; `abcd decide "<text>"` (and `/abcd:decide`) records a decision, `abcd decide` with no operand lists them, and `abcd decide --adr "<title>"` mints an ADR, which bare `abcd decide "<title>"` did before. Accepted cost: an older page or an earlier release's binary still mints an ADR for the bare form until both catch up.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: We expect two pieces of work that each record a decision to land without a collision on decisions. Shown wrong if such a collision happens again after the change lands.
- pursued: decisions recorded on branches from two accounts must merge without anyone resolving a conflict, or working across accounts stalls. Shown wrong if, after this ships, a pull request that records a decision is still dropped from the merge queue over a decision conflict.
