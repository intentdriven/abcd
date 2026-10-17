---
id: adr-2610161733400551
slug: a-reviewer-in-a-model-matrix-run-never
status: proposed
date: 2026-10-16
supersedes: null
superseded_by: null
related_intents: [itd-2610161733316518]
related_rfcs: []
related_adrs: [adr-25]
---

# ADR-2610161733400551: A reviewer or judge in a model-matrix run is told nothing about which model wrote the copy it judges

## Context

itd-2610161733316518 builds one intent several times, once per cell of a matrix of
builder models, and has every copy judged by one or more reviewer models. The comparison
is only as honest as the judging, and the evidence on judging is mixed
([model-matrix SOTA note](../../research/notes/2026-10-16-model-matrix-sota.md)):

- Models recognise their own output from its content, and recognition correlates with
  favouring it (Panickssery, Bowman and Feng, NeurIPS 2024). On verifiable tasks much of
  that preference is legitimate (Chen et al., arXiv 2504.03846), and this repository's
  own note records it as weak or absent under genuine authorship
  ([cross-model review](../../research/notes/2026-07-14-cross-model-review.md)). The bias
  is contested; blinding is cheap insurance, not the main control.
- The same note finds consensus underperforms the best single judge on code: a second
  model is a triage signal, never a vote.
- itd-81 (judge calibration) requires every verdict-rendering agent to be scored on recall
  and on its true-negative rate against a labelled corpus that includes known-good cases.

abcd itself leaks authorship. Every assisted commit carries an `Assisted-by:
<Vendor>:<model>` trailer, and a lane's branch name, pull-request body and run state can
name the model. A reviewer with file and repository access can also read every copy's
branch, because worktrees share one object store. The reading surface already solves the
same problem for cold readings: an include table names what may travel, the assembled
bundle carries no repository path, and a hashed manifest records what was passed
(brief, surface 23).

The open proposal that every review runs as a two-class pair (iss-2610140652230444) asks
for each pair's catch rate to be measured before the rule hardens.

## Decision

We will hand every reviewer in a model-matrix run a self-contained request and nothing
else: the normalised diff of one copy, assembled through the reading surface's include
table and hashed manifest, with commit trailers, author identities, branch names,
pull-request text and run metadata excluded, under an opaque label whose mapping to
builders the run holds and reveals only in the report. A reviewer gets no tools, so it
cannot read the repository, other copies or the mapping. Through a keyed provider this
makes the reviewer a self-contained agent in the provider rule's sense, which is the
condition under which that rule admits it.

We will score each reviewer on its own, never by vote, against three kinds of copy whose
truth is known: copies the hidden tests fail, copies they pass, and copies with defects
seeded on purpose, so recall and the true-negative rate both have a denominator, as
itd-81 asks. A finding the tests cannot confirm or refute is adjudicated once, by a judge
outside the matrix, and reported apart. A pair's catch rate is computed afterwards from
the per-reviewer data as the union of its two reviewers' confirmed catches, matching the
stricter-verdict rule the pair proposal names.

We will report what the blinding does and does not remove: metadata is stripped, content
is not. Each reviewer is also asked, after its verdict, which model it thinks wrote the
copy, and the report gives that guess's accuracy as the measured leak. A model's family
is its developer, the organisation that trained it; verdicts a reviewer gives on a copy
written by its own family are reported apart from the rest.

The same holds for people (product thinker, 2026-10-16). Where a writing task is judged
by a product thinker or a team of them, each judge sees the copies in an abcdesign.app judging
session under random codes, in an order drawn for that judge, rendered through one
template with self-references stripped, and the mapping is revealed only in the report.
People are not asked to guess the author (product thinker, 2026-10-16), so how far they
were kept blind is unmeasured and the report says so; a classifier can name the writing
model from content alone (Sun et al., ICML 2025), so for people too, metadata is
stripped and content is not. A check's findings are judged in two
phases: first alone, shuffled, with no model named and no proposed verdict; then, once
the person's verdicts are locked, beside the proposed verdict and the table of which
models raised what. The copy-to-model mapping never reaches the judging session; where
the host also judges, their verdicts are reported apart as self-blinded.

A verdict is run data. A judging session collects it for the host and writes it to the
run's record on the host's machine; it never reaches GitHub and is never a suggestion
awaiting approval. Guests admitted by code judge only where the host allows it for that
session, and their verdicts are reported apart; every judge is identified by their GitHub
sign-in or a token issued at joining. A copy is rendered as inert text in an isolated
frame, because what a model wrote is untrusted in a page a signed-in judge has open.

## Alternatives Considered

1. **Self-contained, blinded, per-reviewer scoring against known truth (chosen).** Removes
   the authorship signal abcd adds, reuses the reading surface's blinding rather than a
   second copy of it, keeps the no-voting finding, meets itd-81's denominator, and yields
   the pair catch rate. Costs a normalising step, seeded-defect copies, and the loss of
   the file-reading context an ordinary review has: results measure diff-only review.
2. **Reviewers see the copy as the build loop produces it.** No extra step, but every
   verdict is open to authorship cues and to reading other copies.
3. **A panel vote across reviewer models.** PoLL (Verga et al., arXiv 2404.18796)
   favoured panels on general text; the repository's code-specific evidence found
   consensus underperforms the best single judge. Rejected for code.
4. **Pairwise judging of two copies in both orders.** Stronger per judgement
   (CodeJudgeBench, arXiv 2507.10535), but over N copies it costs O(N²) comparisons, and
   hidden tests already give per-copy ground truth. Not adopted; revisit if per-copy
   scores prove too coarse.
5. **Exclude a builder's own family from reviewing it.** Removes the possible bias at the
   cost of fewer reviewers per copy and a confounded reviewer comparison; reporting
   same-family verdicts apart keeps the data.

## Consequences

- A matrix run needs a normalising step between build and review, a held mapping, and a
  set of known-good and seeded-defect copies. All three are part of the run's manifest.
- Reviewer agreement is reported, never used to decide.
- abcdesign.app gains a judging session kind under its access decision
  (adr-2610091016270711); verdicts stay in the host's run record and nothing reaches
  GitHub.
- The ordinary build loop's review is unchanged: this rule binds matrix runs only, which
  is why it is not written as a brief invariant. Whether every review should be blinded
  is a separate decision.
- `Assisted-by:` disclosure is unchanged; the trailer is stripped from what a reviewer
  sees, never from what lands.
