# Adversarial review scales with blast radius

**The rule.** An artefact that changes what gets built crosses its gate only
after independent adversarial review — two reviewers with different lenses
for the major crossings: an intent leaving `drafts/` for `planned/`, an ADR
moving to `accepted`, a plan before execution begins. Artefacts below that
line — ledger captures, comments, routine pull requests — are never blocked
on adversarial review: they are covered by the gates they already have, and
capture in particular must stay frictionless.

**Why.** Both halves are load-bearing, and both are empirical. The 2026-08-19
itd-92 extension went to two independent adversarial reviewers before its
planning interview; both returned NEEDS_WORK, and the findings — a verdict
vocabulary undecidable by the probes it specified, a gate promised with no
acceptance criterion, two trust rules parked in intent prose — would each
have surfaced as implementation failures instead
([the calibration entry](../research/notes/2026-08-15-decomposition-calibration.md)
records the overturn). The floor exists for the same reason the ceiling does:
a blanket review-everything rule would suppress the ledger (recording a
finding must cost less than fixing it, or findings go unrecorded) and would
train the skim that adr-42's warn-storm STOP names — reviewers who read two
mandatory verdicts on trivia skim the one that matters.

**The build-round half.** The rule above scales review to the artefact being
crossed. It says nothing about a build branch reviewed repeatedly as it is
worked, and that omission was where cycle 1 of the cold-reading workstream spent
most of its verification budget. Three rules, measured rather than reasoned
([the cost calibration](../research/notes/2026-08-30-review-round-cost-calibration.md)):

- **A round is reviewed over the delta since its last clean review, never the
  whole branch.** The full-branch pass happens once, at the integration review,
  where reading everything is the point. Repeating it per round re-reads
  already-cleared commits: one branch's fifth pair re-read 42 commits of which
  about 35 had been reviewed four times.
- **A nit does not open a review round.** It is captured and settled at the ship
  commit; only a FIX-FIRST or a BLOCK re-opens the pair. One round returned five
  nits — a misattached godoc, a wrong noun, a loose comment — whose fix round
  would have triggered another pair at roughly 400k tokens to settle five
  sentences. The nits were right; the timing was wrong.
- **The pair is for a trust boundary.** A round touching input parsing, auth,
  secrets, network, subprocess or file/DB access gets both lenses; one that does
  not gets one. This is the repository's existing rule for changes, applied to
  rounds — the place it had never been applied.

None of the three reduces review DEPTH, and that is deliberate: the rounds kept
finding real defects to the last one, including a false claim inside an
already-resolved record. What they reduce is re-reading. Every question about
new code still gets asked, once.

**Measured the same day, and the first rule's justification changed.** Delta
scoping was adopted to save spend and saves 16.8%, against a predicted 50-60%,
even though the diffs shrank from 42 commits to 10 and from 40 to 6. Review cost
here turns out to be dominated by EXPERIMENTATION rather than reading — these
reviewers build harnesses, sweep spellings, run mutation matrices and probe
whole corpora, and that work scales with the surface under test, not with the
diff. What the rule demonstrably buys is FINDING RATE: the delta pairs surfaced
a critical availability defect and a recursive claim defect that four
full-branch passes over the same code had missed, apparently because a reviewer
given six commits reads them and one given forty-two skims. Keep the rule,
justify it on findings, and look to the NUMBER of rounds for cost.

**Bounds.**

- "Independent" is [evaluator-outside-the-loop](evaluator-outside-the-loop.md):
  the reviewers did not produce the artefact, and for the two-reviewer
  crossings their lenses differ (design/feasibility vs record-discipline is
  the proven pair).
- The review is a proposal, never the gate itself
  ([verifier-selects-gates-decide](verifier-selects-gates-decide.md)): the
  human adopts, overrides, or rejects findings — but an unreviewed major
  crossing is a skipped stage, and per [loud-staging](loud-staging.md) a
  skipped stage says so where the crossing is recorded.
- The build-round half is a floor on effort, not a ceiling on judgement: a
  reviewer who finds a FIX-FIRST in the delta may read further back to
  establish whether the class is older, and should. What it forbids is
  re-reading cleared code by default.
- Enforcement is the documented protocol for now (script-first); the
  armed-detector rung — the readiness checks refusing a `drafts/ → planned/`
  move without review receipts in the existing `.abcd/work/reviews/` shape —
  is a recorded seed, built once this protocol has real runs behind it.
