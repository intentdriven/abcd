# Phase 8 — The brief is the shipped state

## Expectation

By the end of this phase, a product thinker opens the brief and knows what
abcd does today, without opening anything else: not the command reference,
not the changelog, not the code. Every surface the binary ships has its
chapter, every chapter says what the surface does and refuses to do, and no
sentence in the brief describes behaviour the binary does not have. The
brief, the record and the release say the same thing about the product,
because a specification cannot close until the brief reflects what it
delivered, and a release cannot be cut while the brief and the binary
disagree.

One state is legitimate and named: a brief edited after a release was cut is
ahead of that release until the next cut. The brief leads and the release
follows, never the other way round, and the gap closes at the next cut
rather than being smoothed over between cuts.

## Milestone

- Every top-level verb and every agent the binary ships has a chapter under
  the brief's surfaces, and the surfaces index lists each with the intent and
  spec that delivered it.
- A spec cannot move to closed while the brief lags the surface it delivered:
  the doc-fidelity pass of
  [itd-60](../../intents/planned/itd-60-doc-fidelity-anti-drift.md) is a hard
  gate at spec close and an advisory after each task.
- The shape claims in the surface chapters are generated from the command
  tree and drift-tested, per
  [itd-147](../../intents/planned/itd-147-the-brief-s-surface-chapters-are-a-generated-reflection-of-t.md);
  the prose around them carries the why and the refusals.
- The brief-to-surface crosscheck gate runs at every release cut against a
  manifest whose pinned inputs name the current brief and the current
  surface, so two honest runs mean the same thing.
- The drift records the v0.7.0 receipts opened are resolved, and the two
  exact duplicates are closed as duplicates.

## Phase Acceptance

- **Given** a product thinker who has read only the brief, **when** they are
  asked what abcd can do at the current release, **then** every capability
  they name exists in the binary and every capability the binary ships is one
  they could have named.
- **Given** a spec whose delivery changed a surface, **when** its intent
  moves to shipped, **then** the brief's chapter for that surface already
  describes the new behaviour, or the move is refused with the lagging
  sentence named.
- **Given** a release cut, **when** the release gate runs, **then** the brief
  and the surface agree, or the cut is refused with the disagreement named.
- **Given** a maintainer edits the brief after a cut, **when** the next cut
  runs, **then** the release reflects the edit and nothing between the cuts
  reported the brief as wrong for being ahead.
- **Given** a shape claim in a surface chapter, **when** the command tree
  changes, **then** the claim changes with it without a hand edit, and a
  hand-written shape claim that disagrees with the tree is refused.

## Scope

Intents bundled:

- [itd-60](../../intents/planned/itd-60-doc-fidelity-anti-drift.md) — the
  doc-fidelity pass: brief and public docs graded against built reality,
  advisory after each task, a hard gate at spec close.
- [itd-147](../../intents/planned/itd-147-the-brief-s-surface-chapters-are-a-generated-reflection-of-t.md)
  — generated surface blocks in the brief's chapters, drift-tested like the
  command reference.

Brief plumbing: the surfaces index under `04-surfaces/`, the release gate
manifest for the brief-to-surface crosscheck, and the surface chapters
themselves.

Ledger records this phase resolves: iss-2609011424033149 and
iss-2609011424033603 (the v0.7.0 verb and agents the brief did not document;
the twenty-one drifted sentences), iss-2608231346137587 (the surface chapters
drift steadily and the gate passes anyway), iss-2609011423385217 (the release
gate manifest's pinned inputs are stale), iss-130 (manifest deletion disarms
the crosscheck), iss-265 (a crosscheck claim without backing).

## Maps against

- [adr-5](../../decisions/adrs/0005-brief-is-current-state.md): the brief is
  the current state of the project. This phase is what makes that decision
  enforceable rather than aspirational, and it adds the one legitimate lead:
  a brief edited after a cut is ahead of the release until the next cut.
- The surfaces index and every chapter under `04-surfaces/`; the invariants
  chapter where a surface's refusals are stated.
- The reverse direction, a human edit to the brief drawing out the intents it
  implies ([itd-61](../../intents/drafts/itd-61-brief-change-derivation.md)),
  is adjacent and stays out of this phase; see the open questions.

## Dependency rationale

Runs after Phase 7: the cold-reading instrument's detection position is
what reads the shipped tree against the record, and the release gate's
crosscheck is the deterministic half of the same question. Runs before the
closed loop (Phase 9, on the design branch): a product thinker's own surface
is only worth building on a brief that tells them the truth.

## Open questions

1. Whether itd-61, the reverse direction, belongs in this phase or in its
   own. The maintainer left it adjacent on 2026-09-01.
2. Settled 2026-09-01: no marker. The release is the boundary; the brief
   always reads as the current design, and a reader at a release reads the
   brief at that tag.
3. Settled 2026-09-01: one mechanism at two points. The shipped move refuses
   per intent and the release cut refuses over everything; the deterministic
   layer needs no oracle and the semantic layer fails closed without one.
