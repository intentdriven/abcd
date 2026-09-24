---
id: itd-2609151138388536
slug: append-only-logs-conflict-on-every-merge-in-a-managed-repo
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
related_issues: [iss-2609100507439414]
origin: extracted-from-record
production_mode: hand-written
related_adrs: [adr-2609151138420062]
---

# Every Decision Is Its Own Record, and the Log Stops Conflicting

Typed links: `related_adrs` [adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted-records-not-one-appe.md) (the shape rule this intent enacts, taken in the same change); `refines` [adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md) (the timestamp-numeric id seam, which this adds a sixth family to and changes in no way); `related_issues` [iss-2609100507439414](../../../work/issues/open/iss-2609100507439414-append-only-logs-conflict-on-every-merge-in-a-managed-repo.md) (the negative finding: the append-only log conflicted on the first two of 27 branch merges). Prose cross-references, not typed links, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)): [itd-2609150819432059](../planned/itd-2609150819432059-abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep.md), the companion that closes the OTHER conflict class — this intent refines nothing of its, and it refines nothing of this intent's, but the two together close both classes and neither closes both alone; and [iss-2609100508570803](../../../work/issues/open/iss-2609100508570803-the-record-verbs-worked-from-worktrees-throughout-the-run.md), the positive finding from the same run that makes this a shape question — five one-file-per-entry record families, zero conflicts, same 27 merges.

## Press Release

> **abcd's decision log becomes a folder of minted records, and the merge
> conflict it produced on every parallel branch goes away.** A decision is no
> longer a line appended to the bottom of one shared file. It is a record with
> an id, allocated through the same seam that mints captures, intents, specs and
> ADRs, written to its own file under `.abcd/work/decisions/`. An index is
> assembled from those files, and `.abcd/work/DECISIONS.md` becomes a symlink to
> it, so every link, every reader and every citation that names the log today
> keeps resolving. Two branches that each record a decision now merge the way two
> branches that each capture an issue already do: git sees two new files and has
> nothing to reconcile. The same shape lands in every repository abcd manages,
> at adoption, rather than being rediscovered and hand-patched there.
>
> "I landed twenty-seven branches in a day and the decision log fought me on the
> first two merges — I ended up hand-writing a merge attribute into
> `.gitattributes` at two in the morning to make it stop," said Maya,
> autonomous-development practitioner. "The issue ledger never once did that, and
> nobody had to think about why. Now the decisions behave like the issues do."

## Why This Matters

Graduated from [iss-2609100507439414](../../../work/issues/open/iss-2609100507439414-append-only-logs-conflict-on-every-merge-in-a-managed-repo.md):
a managed repository's shared append-only files conflicted on nearly every merge
across an autonomous run that landed 27 worker branches through one integration
branch in a single day. `.abcd/work/DECISIONS.md` conflicted on the first two
merges and would have conflicted on every later one; the session stopped it by
hand, by adding a `merge=union` attribute for that path to `.gitattributes`.

The positive finding from the same run,
[iss-2609100508570803](../../../work/issues/open/iss-2609100508570803-the-record-verbs-worked-from-worktrees-throughout-the-run.md),
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
[itd-2609150819432059](../planned/itd-2609150819432059-abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep.md).
This intent `refines` nothing of itd-2609150819432059's, and itd-2609150819432059
`refines` nothing of this one's: they are disjoint. Together they close both
conflict classes, and neither closes both alone. The typed link is a companion
link, recorded so that a reader who finds one half does not conclude the other
half was overlooked.

The shape rule this delivers is
[adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted-records-not-one-appe.md).
The id seam it mints through is
[adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md),
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
harder to read end to end than a file, and **we expect the assembled index plus
the `DECISIONS.md` symlink to recover that** — because the index is the file the
reader had before, and the symlink means no existing reader, link or citation
learns a new path. If a reader has to open 330 files to answer a question the
single file answered, the index has failed and the claim is wrong.

## Scope Conditions

- **Repositories using git.** The claim is about git's merge behaviour on file
  boundaries; nothing here is asserted about another version-control system.
- **abcd itself and every repository abcd manages.** The shape is not optional
  per repository: a managed repository that keeps the single file keeps the
  conflict, and the decision that occasioned this record applies to both.
- **Parallel branch work.** The benefit is proportional to concurrency. At one
  branch at a time the single file never conflicts and the folder buys nothing
  but consistency with the other five families.
- **A filesystem with symlinks, for the interim `DECISIONS.md` form only.** The
  folder and its index do not depend on symlink support; the compatibility
  shim does, and a platform without it reads the index at its own path.
- **Decisions that are independent and dated.** This is the same condition the
  union attribute silently assumed. A decision family where one entry amends
  another in place is outside this claim — an amendment is a new record, on the
  standing rule that a durable record is superseded rather than edited.

## Acceptance Criteria

> _BDD (the [itd-1](../disciplines/itd-1-acceptance-gates.md) discipline)._

- **Given** a repository on the new shape, **when** a decision is recorded,
  **then** it is written as one file under the decisions folder carrying an id
  allocated through the adr-45 seam, and no other committed file is appended to
  by that act.
- **Given** a decisions folder holding entries, **when** the index is assembled,
  **then** the index lists every entry in dated order with its id, and
  `.abcd/work/DECISIONS.md` resolves to that index — so a reader following any
  existing link or citation to `.abcd/work/DECISIONS.md` lands on the assembled
  history and not on a missing file.
- **Given** two branches cut from the same base that each record one decision,
  **when** both are merged into the integration branch, **then** both merges
  complete with no conflict and with both decision files present, and this holds
  with no `merge=union` attribute and no merge driver configured for the path.
- **Given** a repository adopting abcd, **when** adoption runs, **then** that
  repository gets the decisions folder, the assembled index and the compatibility
  form — the same shape abcd itself carries, rather than a merge attribute
  scaffolded into its `.gitattributes`.
- **Given** the new shape is in force, **when** the repository is inspected,
  **then** the decisions-append gate (`scripts/check-decisions-append.sh`, rules
  DA001–DA004) is gone, no CI job invokes it, `make preflight` does not run it,
  and nothing in the record or the conventions names it as a live gate — a folder
  of minted files has no mid-file position to police, so the gate is retired by
  the shape rather than disabled.
- **Given** the existing log's 330 entries, **when** the migration has run,
  **then** every entry is present as its own file with its original date and text
  preserved verbatim, the index renders them in their original order, and the
  count matches — a migration that silently drops or reorders history fails this
  criterion.

## Open Questions

- **Where the folder lives, and what its records are called.** `.abcd/work/` is
  the working tier that holds the log today and the issue ledger beside it, so
  `.abcd/work/decisions/` is the obvious home; whether the family gets its own
  prefix or reuses an existing one is a naming decision the shape does not
  settle.
- **Whether the index stays a symlink or becomes an assembled file.** The
  decision taken is the symlink for now, with a later intent free to assemble a
  single file automatically. If it ever does become a committed assembled file,
  the mechanism claim above says what it must not do: be appended to by the mint.
- **How the migration lands in a repository abcd manages but did not create.**
  abcd rewrites its own log in one change it controls. A managed repository's log
  is the user's file, and a tool that rewrites it unasked is the act
  [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
  tells this project to think twice about. Whether adoption migrates, offers, or
  only applies the shape to new decisions is open.
- **What happens to the union attribute already in `.gitattributes`.** abcd's own
  entry for `.abcd/work/DECISIONS.md` (from
  [iss-118](../../../work/issues/resolved/iss-118-decisions-acknowledgements-multi-writer-merge-hotspot.md))
  becomes dead once the path is a symlink to an index. Removing it is trivial;
  doing it in the same change is the question, since a stale attribute on a path
  that no longer accumulates is harmless and a half-migrated tree is not.
- **Whether `CHANGELOG.md`'s union attribute should go at the same time.** It is
  the same file, a different conflict class, and itd-2609150819432059's business.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
