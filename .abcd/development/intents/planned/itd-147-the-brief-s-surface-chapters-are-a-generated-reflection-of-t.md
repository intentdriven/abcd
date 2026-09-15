---
id: itd-147
slug: the-brief-s-surface-chapters-are-a-generated-reflection-of-t
spec_id: spc-2609020906356450
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
---

# The brief's surface chapters are a generated reflection of the shipped surface, so a shape claim cannot drift

## Press Release

> _Sequenced 2026-09-01 as a delivery rung of
> [Phase 8 — the brief is the shipped state](../../roadmap/phases/phase-8-brief-currency.md);
> the phase records the sync rule and its one legitimate lead._


> **The design record stops being able to lie about the shipped surface.** Each
> surface chapter in the brief carries a generated block — the verb's flags,
> sub-verbs, exit codes, schema fields and counts, derived from the command tree
> at build time and drift-tested like the CLI reference already is. The prose
> around it keeps doing what only prose can: saying why the surface exists, what
> it refuses to do, and which trade was made. A shape claim can no longer drift,
> because nobody writes one by hand.
>
> The measurement that prompted this is uncomfortable. A full-tier
> brief-to-surface crosscheck at the 0.6.2 release gate returned 147
> discrepancies across 23 chapters. Of those, exactly one touched a file the
> release bundle carries. The rest were the record describing a product that had
> moved: a PATH entry documented as a symlink when it has been an owned copy
> since spc-35, a user-scope directory tree missing two of the three directories
> that actually exist, a detection pass enumerating twelve steps where fourteen
> run.
>
> The same day, the same reviewers found **zero** discrepancies across the 775
> lines of `docs/reference/cli/commands.md`. That file is generated and
> drift-tested. That is the whole argument.
>
> "I read the brief to learn what the tool does before I touch it," said Maya, an
> autonomous-development practitioner whose agents read the record the same way.
> "When a chapter says the PATH entry is a symlink and the binary writes a copy,
> I do not discover a documentation bug. I write code against a surface that does
> not exist, and the mistake looks like mine."
>
> "We already knew how to fix this and had done it once," said Kira, who
> maintains the surface. "The CLI reference is generated, so it is right. The
> brief is typed, so it rots. We were keeping two copies of the same fact and
> hand-maintaining the one nobody could check."

## Why This Matters

The brief drifts steadily, not in bursts. Of 124 findings resolvable to a
blaming commit, 66 were last written in one month and 58 in the next — an even
rate, in every class. Roughly half the false claims survived four releases, each
of which recorded a PROMOTE receipt against this same detector at 26, 28 and 37
findings.

So the gate has been running and passing while the record got worse. That is the
proxy-gate class (iss-2608230847432286) with a demonstration attached, and the
deterministic half makes it sharper: `surface_coverage` blocked a commit during
this very release for a missing `history drain` **table row**, and passed green
while false prose claims sat beside those rows. Row presence stands in for
chapter correctness.

The diagnosis is that a chapter mixes two kinds of content with two different
failure modes in the same paragraphs:

- **Shape** — flags, sub-verbs, exit codes, schema fields, counts, file layouts.
  Derivable from the command tree, therefore checkable, therefore never worth
  hand-writing. `false-claim` and `stale-count` are shape by definition, and most
  `undocumented-surface` is the binary growing something a chapter's shape
  section never learned.
- **Intent** — why a surface exists, what it refuses to do, which trade was
  made. Not derivable, and structurally unable to drift against code, because it
  is not a claim about code shape.

Two copies of one fact, one of them authoritative: `one-canonical-primitive`
violated at two copies rather than three.

## Decisions (grilled 2026-09-01)

The maintainer resolved the open questions at the planning interview; these are
commitments, not options.

- **One generated appendix per chapter.** The block sits at the end of the
  chapter under a marker; the prose above it never states a flag or sub-verb.
- **A staged chapter gets an empty block that says unbuilt.** Every chapter
  carries a block; one whose surface has not shipped reads that there is no
  shipped surface, so the reader learns it from the same place as the flags.
- **Flags and sub-verbs only, first.** Exit codes and JSON output fields stay
  prose until the binary records them somewhere a generator can read.
- **The external-state category stays out.** The two-way split (derivable from
  the tree, or rationale) holds; a claim about external state is neither and is
  not this intent's to check.
- **Sequenced as Phase 8**, the brief is the shipped state, with itd-60 as the
  other rung; standalone kind, its own spec.

## What's In Scope

- **One generated appendix per chapter.** Each surface chapter carries a single
  generated block at its end, under a marker. The block is derived from the same
  command tree that already produces `docs/reference/cli/commands.md` and
  `.abcd/development/release/surface.json`, and a drift test fails when the
  committed block and the tree disagree.
- **Flags and sub-verbs, first.** The block carries the verb's flags and its
  sub-verbs, because those are what the command tree records today. Exit codes
  and JSON output fields stay prose until the binary records them somewhere a
  generator can read; the seam is built so admitting them later is a generator
  change, not a re-cut.
- **Every chapter carries a block, including a staged one.** A chapter whose
  surface has not shipped gets an empty block that says so — there is no shipped
  surface — so a reader learns the surface is unbuilt from the same place they
  read the flags, rather than from the block's absence.
- **A marker that keeps prose hand-written.** The generated region is delimited,
  so a chapter's rationale above the marker is never machine-authored and never
  clobbered, and the prose above the marker states no flag and no sub-verb.
- **The trust rule as an ADR plus a brief invariant:** shape claims are derived,
  never hand-authored.
- **Retiring the hand-written shape prose** the generated block replaces, chapter
  by chapter, so the 124 findings are closed by construction rather than by a
  sweep that starts rotting the next day.
- **`surface_coverage` stays, and says what it is.** It remains the row-level
  presence check over the surfaces index, labelled as such, so nothing reads it
  as a chapter-correctness gate.

## What's Out of Scope

- **A claim about external state.** A third category surfaced while this was
  drafted and fits neither half of the split: a claim that is checkable in
  principle but has no local source to check against — the worked example is
  `wrangler.jsonc` recording a dashboard setting with no durable local home,
  where eight of eight pull requests on 2026-08-23 built through an integration
  the record says is off. It is the most dangerous of the three shapes, because
  the plan to change the external state and the assertion that it had changed
  were written by the same hand in the same file, and it reads as a completed
  decision rather than as an unchecked claim. Neither generation nor prose
  reaches it. The two-way split stands for this intent; the category is held by
  iss-2608231607594913 (refining iss-2608220150157502) and a remedy that reads
  the real external state is that record's to propose.
- **A "you changed a surface, so touch its chapter" gate.** It forces an edit
  without forcing correctness, which is a phantom gate — worse than none, and the
  precise class this intent is an instance of.
- **A new principle.** `one-canonical-primitive` already says it; this is an
  application, and the confirmed decomposition records "none new" deliberately.
- **Generating the rationale.** Prose about why is the half that cannot drift and
  the half worth a human. This intent therefore does not make a hand-maintained
  behavioural claim safe: ordering guarantees, failure semantics and what a verb
  refuses stay prose and stay hand-maintained.
- **Fixing the 147 by hand ahead of the seam.** The corpus is evidence; a hand
  sweep destroys the dataset and buys a clean brief that drifts again at the
  measured rate.

## Mechanism

> _Facilitator-seeded from the measurement this intent rests on. The maintainer
> stated no mechanism claim at the interview; strike this line and the paragraph
> under it if it is not the claim being made._

We expect this to work because it has already worked here, once, on the same
class of content: `docs/reference/cli/commands.md` is generated from the command
tree and pinned by `TestReferenceMatchesCommittedPage`, and it returned zero
discrepancies on the day the hand-written brief returned 147. The mechanism needs
no new infrastructure — the generator, the snapshot and the drift test all exist
and run every CI pass — and a `false-claim` or `stale-count` finding about a
generated block in the next crosscheck run is what would show it wrong.

## Scope Conditions

None stated.

## SOTA

Doc-generation from a command tree is the presumptive answer and is what cobra
ecosystems do by default. The adversary-filtered question is not whether to
generate but where to cut, because a fully generated chapter loses the rationale
that makes the brief worth reading. The cut proposed here — generated shape,
hand-written why — is the narrow version, chosen because the measurement says
shape is where the drift is.

## Acceptance Criteria

- **Given** a surface chapter with a generated appendix under its marker,
  **when** a verb gains a flag or a sub-verb and the appendix is not regenerated,
  **then** the drift test fails and names the chapter and the missing claim.
- **Given** a chapter's hand-written rationale above the marker, **when** the
  appendix is regenerated, **then** nothing above the marker changes.
- **Given** a chapter whose surface has not shipped, **when** the generator runs,
  **then** the chapter carries an empty block under the same marker stating there
  is no shipped surface, so the reader learns the surface is unbuilt from where
  they would have read the flags.
- **Given** the generated appendix, **when** it is composed, **then** it carries
  the verb's flags and sub-verbs and nothing else — an exit code or a JSON output
  field stays prose until the binary records it where a generator can read it.
- **Given** a surface chapter's prose above the marker, **when** it is checked,
  **then** it states no flag and no sub-verb; the shape claims live only in the
  appendix.
- **Given** a full-tier brief-to-surface crosscheck run after the seam lands,
  **when** its findings are classified, **then** no finding is a `false-claim` or
  `stale-count` about a flag or a sub-verb the generated appendix covers.
- **Given** `surface_coverage`, **when** it reports, **then** it names itself as
  the row-level presence check over the surfaces index and claims nothing about
  whether a chapter's prose is correct.
- **Given** the crosscheck's non-reproducibility (iss-2608231409595789),
  **when** progress is assessed, **then** it is assessed on which chapters are
  implicated rather than on a finding count, because the count is not a metric.

## Open Questions

> _Every question below was put to the maintainer at the planning interview on
> 2026-09-01 and answered there; the answers are the rulings in the Decisions
> section above. The section is kept so a later reader sees what was asked._

- **Resolved — where the seam is.** One generated appendix per chapter, at the
  end, under a marker. Not per section: the per-section cut preserves the most
  rationale and costs the most machinery, and the appendix keeps the prose
  hand-written for a fraction of it.
- **Resolved — whether the snapshot carries enough.** Flags and sub-verbs only,
  first. Exit codes and JSON output fields are the gaps, and they stay prose
  until the binary records them somewhere a generator can read, rather than being
  invented into the snapshot to fill the block.
- **Resolved — a chapter documenting a staged surface.** It gets an empty block
  that says there is no shipped surface. Every chapter carries a block, so a
  reader never has to infer anything from a block's absence.
- **Resolved — what happens to `surface_coverage`.** Left as the row-level check
  it is, and labelled as that. Leaving it was always defensible; leaving it
  unlabelled is what let it read as a chapter-correctness gate.
- **Resolved — the third, external-state category.** Recorded as out of scope
  above. The two-way split stands for this intent, and the category is held by
  iss-2608231607594913, refining iss-2608220150157502.

## Audit Notes

Filed from `iss-2608231346137587`, whose Routing section carries the four-piece
decomposition confirmed by the maintainer on 2026-08-23 (verdict SPLIT), graded
into the dated decomposition-calibration corpus. Capability here; trust rule to
an ADR plus a brief invariant; stance deliberately none new; plumbing to the
brief.

The evidence base is three full-tier crosscheck runs preserved out-of-tree with
their content commits, manifest hash and tier. Read it with
`iss-2608231409595789` in hand: the runs returned 125, 126 and 147, and runs 2
and 3 are a controlled comparison — the brief is byte-identical between their
commits, so the detector's whole subject was held constant and it still returned
17% more findings. Which chapters are implicated is broadly stable; how many and
of what class is not.

## Grounds

- pursued: a shape claim nobody writes by hand cannot drift; a false-claim or stale-count finding about a generated block in the next crosscheck run shows it was wrong
