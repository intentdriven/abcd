---
id: adr-2609151138420062
slug: the-decisions-log-is-a-folder-of-minted-records-not-one-appe
status: accepted
date: 2026-09-15
supersedes: null
superseded_by: null
related_intents: [itd-2609151138388536]
related_rfcs: []
related_adrs: [adr-45]
---

# ADR-2609151138420062: The decisions log is a folder of minted records, not one appended file

## Context

An autonomous-run field experiment in a managed repository on 2026-09-09/10
landed 27 worker branches through one integration branch in a single day. It
produced a clean natural experiment on file shape, because the same day, the
same branches and the same merges exercised two different ways of storing a
growing record.

**The two files that conflicted are the two single, append-to-the-bottom files.**
`.abcd/work/DECISIONS.md` conflicted on the first two merges and would have
conflicted on every later one; the session stopped it by adding a `merge=union`
attribute for that path to `.gitattributes` by hand, and after that the log did
not conflict again. `CHANGELOG.md`'s `[Unreleased]` section has the same shape
and conflicted on four of the eight merges that carried an entry; union is not
safe there, because it duplicates the `###` headings, so that half needed a
hand-written section-merging script kept only in the session's local tier. Both
are recorded in
[iss-2609100507439414](../../../work/issues/open/iss-2609100507439414-append-only-logs-conflict-on-every-merge-in-a-managed-repo.md).

**The five record families that never conflicted are all one file per entry.**
The same run resolved 33 issues, shipped 5 intents, superseded 3 ADRs, held 3
intents with stated reasons and recorded 1 ideate verdict, every worker in its
own git worktree. Across all 27 merges the record store's move operations —
`open/` to `resolved/` and the rest — did not conflict once. That is the
measurement in
[iss-2609100508570803](../../../work/issues/open/iss-2609100508570803-the-record-verbs-worked-from-worktrees-throughout-the-run.md),
filed as a positive finding rather than a defect, and it is what turns this from
a plumbing question into a shape question. Two storage shapes, one day, one set
of branches: the one-file-per-entry families produced zero conflicts and the
single-file logs produced conflicts on nearly every merge. Nothing about the
content differs — a decision line and an issue record are both short, dated and
independent. What differs is whether two concurrent writers touch the same
region of the same file.

The log itself anticipated this. `.abcd/work/DECISIONS.md`'s header has carried
the instruction since it was written: "Graduate this file to per-file
`decisions/<date>--<slug>.md` if size or parallel-agent merge contention bites."
It has now bitten, measured, at 330 entries and 2,489 lines.

Three constraints were already locked before this decision. Record ids are
minted through one timestamp-numeric allocator that reads no maximum
([adr-45](0045-record-ids-are-timestamp-numeric-and-capture-stable.md)), which
is what lets two checkouts mint in the same window without coordinating — and
since the ruling of 2026-09-01 that allocator serves ADRs too, so a sixth family
is a caller of an existing seam and not a new mechanism. The record
information architecture already treats folder membership as the status signal
for a family stored one file per record
([adr-30](0030-record-information-architecture.md)). And the decisions-append
gate `scripts/check-decisions-append.sh` (DA001–DA004) exists precisely because
a single appended file has a position to police: an entry inserted mid-file, a
historical entry reworded, a merge that authors a line no parent had.

What forces the decision now is that the workaround does not travel and does not
scale as a stance. The union attribute is a per-repository `.gitattributes`
entry that abcd would have to write into a repository it adopts, on a path that
repository's user owns; it makes the interleaving of two concurrent appends
non-deterministic; and it is correct only while every entry is independent and
dated, a property nothing checks and nothing enforces. Scaffolding it into
somebody's repository unasked is the act this project's own rule about
user-owned space
([adr-2609091248200336](2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md))
tells it to think twice about. The shape needs no attribute anywhere.

## Decision

**A decision is a record, minted like every other record, stored one per file.**
The decisions log is a folder. Each entry is a file carrying an id allocated
through the adr-45 seam — the same allocator captures, intents, specs and ADRs
already mint through — with its date and its text. Recording a decision creates
a file and appends to nothing.

**An index is assembled from the folder, and it is derived, never appended to.**
The index lists every entry in dated order with its id, so the reading experience
the single file gave is recovered. It is produced from the folder's contents; the
minting verb does not write to it. A committed index that every mint also
appends to would move the conflict rather than remove it, and that is the one
construction this decision forbids by name.

**`.abcd/work/DECISIONS.md` becomes a symlink to that index for now.** Every
existing reader, link, citation and convention that names the path keeps
resolving, and nothing has to be rewritten in one sweep to make the shape land.
A later intent may assemble a single file automatically if that reads better; the
symlink is the compatibility form, not the end state, and the decision does not
pre-empt that choice.

**This applies to abcd itself and to every repository abcd manages.** A managed
repository does not get the single file plus a merge attribute scaffolded into
its `.gitattributes`. It gets the shape, at adoption, the same shape abcd
carries.

**The decisions-append gate is retired by the shape, not disabled.** DA001–DA004
check position, preservation, merge authorship and byte-cleanliness inside one
appended file. A folder of individually minted files has no mid-file position to
insert into: an entry is added by creating a file, a historical entry is
protected by the same durability rule every other record family lives under, and
a merge that authors a line no parent had is not a shape git can produce from two
disjoint file additions. The gate goes when the shape lands, in the same change,
and nothing is left invoking it.

The delivery is
[itd-2609151138388536](../../intents/drafts/itd-2609151138388536-append-only-logs-conflict-on-every-merge-in-a-managed-repo.md).
This record is the rule; that intent is the capability, with the migration, the
index assembly, the adoption path and the gate retirement as its acceptance
criteria.

## Alternatives Considered

1. **Keep the single file and propagate the union merge attribute to every
   managed repository.** The cheapest option, and the one the field session
   actually performed: a one-line write into a file adoption already touches, and
   it demonstrably stopped the conflict for the remaining 25 merges. Rejected on
   three grounds. It writes into a file the user owns, on a repository abcd
   adopts rather than creates, which is the act adr-2609091248200336 is about. It
   is correct only under an unchecked precondition — that entries are independent
   and order-insensitive — so the day somebody writes an entry that amends
   another, union merges both silently and nothing notices. And it buys nothing
   for the changelog, the other half of the same finding, where union is actively
   wrong because it duplicates headings: a remedy that closes one of two
   identical-looking classes and leaves the other is worse than one that names the
   difference.

2. **The folder for abcd only, while managed repositories keep the file and the
   attribute.** Attractive because abcd controls its own history and can migrate
   in one change, while a managed repository's log is somebody else's file.
   Rejected: it makes the tool's own conventions and the conventions it ships two
   different things, which is the failure mode this project keeps catching in
   other forms — abcd had already answered both halves of this finding for itself
   and neither answer travelled, and that non-travelling is exactly what
   iss-2609100507439414 is a report of. A rule that holds only where the author
   works is not a rule.

3. **Keep the single file and accept the conflicts.** Defensible at one branch at
   a time, where the file never conflicts and the folder buys only consistency.
   Rejected: the measurement is from parallel work, and parallel work is the
   direction of travel, not an edge case. The cost is not the conflict itself but
   what a conflicted append-only ledger invites a session to do at two in the
   morning — resolve it by hand, drop a side, or reorder history — against a gate
   whose whole purpose is to refuse exactly that.

4. **Chosen: a folder of individually minted decision records, with a derived
   index and `DECISIONS.md` as a symlink to it, in abcd and in every repository
   it manages.** It removes the conflict class rather than patching it, needs no
   `.gitattributes` entry anywhere, puts decisions on the same footing as the five
   families that were measured not to conflict, and retires a gate instead of
   adding one.

## Consequences

- **Two branches recording a decision stop conflicting, with no configuration.**
  Git sees two file additions and reconciles them without help. Nothing has to be
  written into `.gitattributes`, in abcd or in any repository it adopts, and the
  interleaving question disappears rather than being answered
  non-deterministically.
- **Decisions join the record families and inherit their tooling.** An id, a
  file, a folder — the same shape the citation resolver, the `abcd <record-id>`
  dispatch, the lifeboat packer and the record gates already read. A sixth family
  costs the adr-45 seam nothing, which is what a seam is for.
- **Reading the whole history gets harder before the index makes it easier
  again.** A single file is grep-able, scroll-able and diff-able in one pass; 330
  files are not. The assembled index is what recovers that, and it is now
  load-bearing rather than a convenience — if the index is absent, stale or
  unassembled, the decision history is materially less readable than it was
  before this change. That is the real cost of the decision, and it is paid in
  a piece of machinery that has to work.
- **Every existing reference to `.abcd/work/DECISIONS.md` keeps resolving —
  through a symlink, which is a new dependency.** `AGENTS.md`, the brief, the ADR
  README and many records name that path. The symlink means none of them changes
  on the day the shape lands. It also means a platform or a tool that does not
  follow symlinks now sees something different from what it saw before, and that
  the interim form is one more thing to remove when the end state is chosen.
- **The decisions-append gate goes, and with it a body of careful work.**
  `scripts/check-decisions-append.sh` and its case suite encode four rules and a
  long argument about why position alone is not append-only. Retiring them is a
  net simplification, and it is also the deliberate loss of a check nothing
  replaces: after this change, a decision record's immutability rests on the same
  convention every other record family rests on — a durable record is superseded,
  not edited — with no gate behind it. If that convention needs mechanical
  enforcement, it needs it for all six families, not for one file.
- **The existing log's 330 entries have to be migrated, and the migration is the
  risky part.** Each entry becomes a file with an id, its date and its text
  preserved verbatim, in its original order. A migration that drops, reorders or
  reflows history is worse than the conflict it fixes, which is why the intent
  makes verbatim preservation and a matching count an acceptance criterion rather
  than an implementation note.
- **A managed repository's existing log is somebody else's file.** abcd rewrites
  its own in a change it controls. What adoption does to a repository that
  already has a `DECISIONS.md` — migrate it, offer to, or apply the shape only to
  new decisions — is left open in the intent, because the answer is a question
  about how opinionated adoption is allowed to be and not about file shape.
- **The changelog half stays open, and is not closed by this record.**
  `CHANGELOG.md`'s `[Unreleased]` section is the other single append-to-the-bottom
  file, and union is the wrong remedy for it. abcd solved it for itself by
  deriving the changelog from records at the cut and refusing a non-empty
  `## [Unreleased]`; what a managed repository lacks is the ability to use that
  flow, which is
  [itd-2609150819432059](../../intents/drafts/itd-2609150819432059-abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep.md).
  Neither record refines the other; both conflict classes close only when both
  land.
- **The union attribute for `CHANGELOG.md` stays where it is.** This record
  retires the attribute for the decisions path, once that path is a symlink to a
  derived index. The changelog's entry is a different class with a different
  answer and is untouched here.

## Status note

**Accepted by the product thinker's ruling of 2026-09-15**, taken on the two
findings above read together: the decisions log becomes a folder of individually
minted entries with an assembled index, `DECISIONS.md` becomes a symlink to that
index for now, the shape applies to abcd and to every managed repository, and the
decisions-append gate is retired by the shape.
