---
id: spc-2609020906356450
slug: the-brief-s-surface-chapters-are-a-generated-reflection-of-t
intent: itd-147
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The brief's surface chapters carry a generated appendix that cannot drift

## Summary

The spec delivers itd-147: each of the brief's surface chapters carries one
generated appendix, at the end, under a marker. The appendix holds the verb's
flags and sub-verbs, derived from the same command tree that already produces
`docs/reference/cli/commands.md`, and a drift test fails when the committed
appendix and the tree disagree. The prose above the marker keeps the half that
cannot drift — why the surface exists, what it refuses, which trade was made —
and states no flag and no sub-verb.

Every chapter carries a block, including one whose surface has not shipped: that
chapter gets an empty block saying there is no shipped surface, so a reader
learns the surface is unbuilt from the same place they would read the flags,
rather than from a block's absence.

Exit codes and JSON output fields stay prose, because the tree does not record
them where a generator can read them. `surface_coverage` stays exactly as it is
and is labelled as the row-level presence check it is.

## Scope

- **The generator.** One pass over the command tree emits, per surface chapter, a
  block listing the verb's flags and its sub-verbs. It reuses the tree walk that
  backs `cli.GenerateReference` and the surface snapshot, so there is one
  traversal of the command tree in the repository and not a second one that can
  disagree with it.
- **The marker.** A delimited region at the end of each chapter — an opening and
  a closing comment the generator owns. Everything above the opening marker is
  hand-written and never touched; everything between the markers is machine
  written and never hand-edited. The generator refuses a chapter whose markers are
  absent, crossed, or duplicated, rather than guessing where the region is.
- **The unbuilt block.** A chapter whose surface has no shipped representation in
  the tree gets the same markers with a single sentence between them stating there
  is no shipped surface. Uniform presence is the point: a reader never infers
  anything from absence.
- **The drift test.** `TestSurfaceBlocksMatchCommandTree` (name provisional),
  built the way `TestReferenceMatchesCommittedPage` is: regenerate in memory,
  compare against the committed chapters, and fail naming the chapter and the
  claim that differs.
- **Retiring the hand-written shape prose.** Chapter by chapter, the flag and
  sub-verb sentences the appendix replaces are removed from the prose above the
  marker, so the measured findings close by construction. A check that the prose
  above the marker names no flag and no sub-verb keeps them from coming back.
- **The trust rule, recorded.** An ADR stating that shape claims in the record are
  derived and never hand-authored, plus the matching invariant in the brief's
  invariants chapter.
- **`surface_coverage`, labelled.** Its report and its rule documentation say what
  it checks: a row exists in the surfaces index for each surface. It claims
  nothing about whether a chapter's prose is correct.

## Approach

The whole argument for this design is that it has already worked here, once, on
the same class of content, so the approach is to copy the shape of that success
rather than invent one: one generator, one committed artefact, one drift test
that fails in CI.

The cut is where the two failure modes separate. Shape — flags, sub-verbs — is
derivable, therefore checkable, therefore never worth hand-writing. Intent —
why, and what the surface refuses — is not derivable and cannot drift against
code, because it is not a claim about code shape. The marker is the physical
form of that cut, and the refusal on a malformed marker is what keeps the two
halves from bleeding into each other silently.

The narrow first cut is deliberate. Flags and sub-verbs are what the tree
records; exit codes and JSON fields are what it does not. Generating a block from
fields nobody records would put invented content under a marker that claims to be
derived, which is worse than the prose it replaced. Admitting them later is a
generator change against a seam that already exists.

## How the Acceptance Criteria are satisfied

- **ac-1 (a new flag or sub-verb fails the drift test).** The test regenerates
  from the tree and diffs against the committed chapters; a fixture tree with an
  added flag and an unregenerated chapter fails, and the failure message names the
  chapter and the claim.
- **ac-2 (regeneration leaves prose alone).** The writer replaces only the bytes
  between the markers. The test takes a chapter with distinctive prose above the
  marker, regenerates, and asserts the prefix is byte-identical.
- **ac-3 (a staged chapter gets an empty block).** A fixture chapter whose surface
  is absent from the tree yields the markers with the unbuilt sentence between
  them. The test asserts the markers are present and the sentence is the declared
  one — not an empty region, and not an omitted block.
- **ac-4 (flags and sub-verbs only).** The block's composer takes flags and
  sub-verbs as its whole input. The test asserts a fixture verb carrying an exit
  code and a JSON schema in its metadata emits neither into the block.
- **ac-5 (prose above the marker states no shape).** A check over every chapter
  reads the region above the opening marker and fails on a flag spelling or a
  sub-verb name. The test runs it against a chapter that carries one and against
  one that does not.
- **ac-6 (no `false-claim` or `stale-count` about a covered claim).** Verified by
  running the full-tier crosscheck after the seam lands and classifying its
  findings. This one is an outcome check, not a unit test, and its result is
  recorded against the intent rather than asserted in CI.
- **ac-7 (`surface_coverage` says what it is).** Its finding text and its rule
  documentation are amended and pinned by a test on the message, so the label
  cannot quietly revert.
- **ac-8 (progress is assessed by chapters implicated).** The assessment note
  records which chapters are implicated across runs. Nothing asserts a finding
  count, because the count is not reproducible (iss-2608231409595789).

## Tests

- Generator table over command-tree fixtures: a verb with flags, one with
  sub-verbs, one with both, one with neither, one absent from the tree.
- Marker tests: absent markers, crossed markers, two opening markers, a marker
  inside a fenced block — each refused by name rather than repaired.
- Idempotence: two consecutive regenerations produce identical bytes.
- Prose-preservation: the region above the marker is byte-identical across a
  regeneration.
- The drift test itself, watched failing against a deliberately stale chapter
  before the generator is wired.
- The no-shape-in-prose check, against a positive and a negative fixture.
- A `surface_coverage` message test pinning the row-level label.

## Out of scope

- **A claim about external state** — checkable in principle, no local source to
  check against. Held by iss-2608231607594913 (refining iss-2608220150157502);
  the two-way split stands for this spec, and a remedy that reads the real
  external state is that record's to propose.
- **A "you changed a surface, so touch its chapter" gate** — an edit forced
  without correctness forced, which is the phantom-gate class this intent is an
  instance of.
- **A new principle.** `one-canonical-primitive` already says it; this is an
  application.
- **Generating the rationale**, and therefore any guarantee about a
  hand-maintained behavioural claim: ordering guarantees, failure semantics and
  what a verb refuses stay prose and stay hand-maintained.
- **Fixing the measured findings by hand ahead of the seam.** The corpus is
  evidence; a hand sweep destroys the dataset and buys a brief that drifts again
  at the measured rate.
- **Exit codes and JSON output fields in the block**, until the binary records
  them where a generator can read them.
