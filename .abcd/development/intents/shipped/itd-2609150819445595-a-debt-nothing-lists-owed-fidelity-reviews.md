---
id: itd-2609150819445595
slug: a-debt-nothing-lists-owed-fidelity-reviews
spec_id: spc-2609202112205096
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
related_issues: [iss-2609100509537730]
origin: extracted-from-record
production_mode: hand-written
impact: additive
---

# A shipped intent's owed fidelity review is listed, so the debt is paid

## Press Release

> **Every shipped intent owes a fidelity review, and abcd now says which ones are still owed.** The close that ships an intent mints the review receipt and marks the intent as owed; until now nothing counted those marks across the record, so a session that wanted to pay the debt enumerated the shipped intents by hand or grepped the decision log for receipt ids. With this change the intent lifecycle board shows the count, a read-only listing names every owed intent with its receipt, and the record dispatcher on a shipped intent says "fidelity review owed" with the command that re-emits the request, instead of "none". Nothing refuses: the close mints the debt in the same change, so a gate on it would block by construction. The listing makes the debt visible; a host still runs the auditor.
>
> "I asked what was outstanding and got a list of eight, with the command to re-emit each one," said Iris, a technical facilitator paying down a run's review debt. "Last week the same question was a grep through the decision log."

## Why This Matters

Twenty receipts were owed in one managed repository during an autonomous run, none ingested, and no status render, lint row or listing said so. On 2026-09-18 a second repository paid twelve in a day and found what was owed by grepping the decision log. abcd's own record owed seven of forty-six shipped on 2026-09-20. The obligation is real and by design; it accrues silently because no verb answers "what is outstanding", and the only way to learn is the enumeration the debt makes expensive. Graduated from `iss-2609100509537730`.

## Mechanism

We expect a count of owed fidelity reviews on the bare status surfaces to be enough to make the debt get paid, because the debt was invisible rather than resisted, and a session that was asked what was outstanding discharged three in one sitting; it is shown wrong if the count is rendered and the debt still accumulates, which would mean visibility was not the constraint.

## Scope Conditions

- Holds where a host runs the auditor on the emitted request; the listing makes the debt visible and pays nothing. <!-- cond: cond-2609202112215043 -->
- Holds for intents shipped through `spec close`, whether or not the review marker existed when they shipped. <!-- cond: cond-2609202112219188 -->

## Decisions

Settled on 2026-09-20. The first two had one defensible answer each and were recorded on the product thinker's ruling of that day that a single genuine answer is recorded, not asked (captured as `iss-2609202055103741`); the third and fourth were asked.

1. **List, never refuse.** Every `spec close` mints an owed receipt in the same change, so a refusal in `lint`, `record-lint` or `launch ship` would block by construction. The debt is listed and dispatched, not gated.
2. **The listing points at the re-emit, not at the request file.** The request path is gitignored and sweepable, so a listing that printed it could name a file that is gone; `abcd intent audit <itd-N>` re-emits it on demand.
3. **The owed set is OWED plus no-marker.** An intent nobody ever reviewed is a debt whether or not a receipt was minted, so a shipped intent with no marker (shipped before markers, or a ship whose receipt failed to mint) is listed as owed until a review is ingested; nothing is grandfathered. A dead-lettered intent is shown under its own heading, unreviewed with the reason, and is not counted as owed.
4. **Three surfaces, together.** Bare `abcd intent audit` is the read-only listing (bare invocations are read-only by convention here); bare `abcd intent` carries the owed count beside its bucket counts; `abcd <itd-N>` on an owed intent says so as its next move. The design review found the three already wired or nearly so, and no surface among them is a gate.

## Acceptance Criteria

- **Given** a shipped intent whose marker reads owed, **when** the listing runs, **then** it names the intent, its receipt id and the re-emit command, and exits 0.
- **Given** a shipped intent whose marker reads ingested, **when** the listing runs, **then** it is not listed as owed.
- **Given** a shipped intent whose marker reads dead-lettered, **when** the listing runs, **then** it appears under its own heading as unreviewed, with the dead-letter reason, and is not counted in the owed total.
- **Given** a shipped intent with no marker at all, **when** the listing runs, **then** it is listed as owed with no receipt, the entry says a receipt is minted on re-emit, and the owed total includes it.
- **Given** bare `abcd intent audit`, **when** it runs, **then** it is the listing above, writes nothing, and exits 0.
- **Given** bare `abcd intent`, **when** it renders, **then** it carries the owed count beside the bucket counts, and the count equals the listing's owed total.
- **Given** `abcd <itd-N>` on a shipped intent whose marker reads owed, **when** it renders, **then** the next move says the review is owed, names the receipt, and names the re-emit command.
- **Given** `--json`, **when** the listing runs, **then** the payload carries one entry per shipped intent with its marker state and receipt, and no path into the local tier.

## Open Questions

_None open; the four decisions above settle the interview's questions._

## Typed Links

- **refines `itd-53`** (a shipped intent no longer drifts out of audit because nobody ran the review): That planned record wants the review to run; this one makes the unrun review visible, which is the smaller rung.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-dd80727d7bd1 -->
Fidelity review — receipt rcp-dd80727d7bd1 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:2856f90ec95400e220ab46f6c9059e79e3362bbeee9a9958e9ab74ed257974c9
Input attestations: diff:internal/core/intent/owed.go, internal/core/record/record.go, internal/surface/cli/cli.go at BASE 4ce401c5 (tree 376356ce669156f45b423eb7858a952ffeb0f9d6); live run of `go run ./cmd/abcd intent audit`, `--json`, bare `intent` and `abcd itd-53` on that tree on 2026-09-26@-;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the live listing on the BASE tree printed 'itd-53 receipt rcp-d2372b1cb47f — re-emit: abcd intent audit itd-53' and exited 0; runOwedReviews renders intent.Reviews with ReEmitCommand per entry, and TestIntentAuditBareListsOwedReviews pins it
  evidence: internal/surface/cli/cli.go:2823 — "func runOwedReviews(cmd *cobra.Command, asJSON bool) error"
  evidence: internal/core/intent/owed.go:66 — "func ReEmitCommand(intentID string) string"
  evidence: internal/surface/cli/intent_owed_test.go:42 — "TestIntentAuditBareListsOwedReviews"
- ac-2 — MET: IsOwed is OWED plus none, so INGESTED is never owed; the live listing counted 96 ingested of 100 shipped and listed none of them; TestReviewsReadsEveryShippedMarker asserts the INGESTED fixture is not owed
  evidence: internal/core/intent/owed.go:52 — "func (e ReviewEntry) IsOwed() bool"
  evidence: internal/core/intent/owed_test.go:101 — "got["itd-12"].IsOwed()"
  evidence: internal/core/intent/owed_test.go:60 — "TestReviewsReadsEveryShippedMarker"
- ac-3 — MET: the CLI prints a 'dead-lettered (unreviewed; not counted as owed):' heading, deadLetterReason reads the reason off the marker line, and the core test asserts the reason and that the dead-lettered fixture is not owed and counted apart (Owed 3, DeadLettered 1)
  evidence: internal/surface/cli/cli.go:2848 — "dead-lettered (unreviewed; not counted as owed):"
  evidence: internal/core/intent/owed.go:129 — "func deadLetterReason(content, rcp string) string"
  evidence: internal/core/intent/owed_test.go:93 — "if r := got["itd-13"].Reason; r != deadReason"
  evidence: internal/core/intent/owed_test.go:106 — "if l.Owed != 3 || l.DeadLettered != 1 || l.Ingested != 1"
- ac-4 — MET: the live listing on the BASE tree carried 'itd-3 no receipt (one is minted on re-emit) — re-emit: abcd intent audit itd-3' inside the owed total of 4; IsOwed includes the no-marker state and the core test asserts the none fixture (itd-14) is owed
  evidence: internal/surface/cli/cli.go:2844 — "no receipt (one is minted on re-emit) — re-emit: %s"
  evidence: internal/core/intent/owed.go:57 — "Owed counts OWED plus none"
  evidence: internal/core/intent/owed_test.go:101 — "!got["itd-14"].IsOwed()"
- ac-5 — MET: bare `go run ./cmd/abcd intent audit` on the BASE tree rendered the listing, exited 0 and left git status unchanged; the reader is documented and tested as never writing
  evidence: internal/core/intent/owed.go:14 — "It reads and never writes: it does not re-emit a"
  evidence: internal/core/intent/owed_test.go:132 — "TestReviewsNeverWrites"
  evidence: internal/surface/cli/cli.go:2833 — "abcd intent audit — fidelity reviews owed %d · dead-lettered %d · ingested %d (of %d shipped)"
- ac-6 — MET: bare `intent` on the BASE tree rendered '… superseded 13 · reviews owed 4', equal to the listing's owed 4; TestStatusCountsOwedReviews asserts ReviewsOwed equals the listing's Owed and TestIntentBareCarriesOwedCount pins the render
  evidence: internal/surface/cli/cli.go:2160 — "· superseded %d · reviews owed %d"
  evidence: internal/core/intent/owed_test.go:160 — "if v.ReviewsOwed != l.Owed || v.ReviewsOwed != 3"
  evidence: internal/surface/cli/intent_owed_test.go:125 — "TestIntentBareCarriesOwedCount"
- ac-7 — MET: `go run ./cmd/abcd itd-53` on the BASE tree rendered 'next: fidelity review owed, receipt rcp-d2372b1cb47f — re-emit the request with `abcd intent audit itd-53`'; record.go composes that move and the record and CLI tests assert it
  evidence: internal/core/record/record.go:343 — "fidelity review owed, receipt " + r.ReceiptID + " — re-emit the request with `" + r.ReEmit + "`"
  evidence: internal/core/record/record_test.go:769 — "want: []string{"fidelity review owed", "rcp-0000000000a1", "`abcd intent audit itd-4`"}"
  evidence: internal/surface/cli/intent_owed_test.go:147 — "TestRootDispatchNamesAnOwedReview"
- ac-8 — MET: `intent audit --json` on the BASE tree returned entries for all 100 shipped intents, each with intent_id, state and receipt_id, and the serialised payload contained no .work.local path; TestReviewsCarriesNoLocalTierPath and TestIntentAuditBareJSON pin both
  evidence: internal/core/intent/owed.go:43 — "under the gitignored local tier; that path is never carried here"
  evidence: internal/core/intent/owed_test.go:114 — "TestReviewsCarriesNoLocalTierPath"
  evidence: internal/surface/cli/intent_owed_test.go:86 — "TestIntentAuditBareJSON"

Gap audit:
- honoured:
  - the lifecycle board shows the owed count
    evidence: internal/surface/cli/cli.go:2160 — "reviews owed %d"
  - a read-only listing names every owed intent with its receipt and the re-emit command
    evidence: internal/surface/cli/cli.go:2823 — "func runOwedReviews("
    evidence: internal/core/intent/owed_test.go:132 — "TestReviewsNeverWrites"
  - the record dispatcher on a shipped intent says fidelity review owed with the re-emit command instead of none
    evidence: internal/core/record/record.go:343 — "fidelity review owed, receipt"
  - the owed set is OWED plus no-marker; dead-lettered is shown apart and not counted (decision 3)
    evidence: internal/core/intent/owed.go:57 — "Owed counts OWED plus none"
    evidence: internal/surface/cli/cli.go:2848 — "dead-lettered (unreviewed; not counted as owed):"
  - the listing points at the re-emit, never at the gitignored request path (decision 2)
    evidence: internal/core/intent/owed.go:43 — "that path is never carried here"
- diverged: (none)
- missing: (none)

Scope-condition dispositions:
- cond-2609202112215043 — survived: the reader lists and never writes or pays; this audit itself is a host running the auditor on the re-emitted request, exactly the arrangement the condition assumes
  evidence: internal/core/intent/owed.go:14 — "It reads and never writes"
  evidence: internal/core/intent/owed_test.go:132 — "TestReviewsNeverWrites"
- cond-2609202112219188 — survived: a shipped intent with no marker at all (itd-3 on the BASE tree, shipped before markers) is listed as owed with no receipt, so the listing holds for intents shipped before and after the marker existed
  evidence: internal/surface/cli/cli.go:2844 — "no receipt (one is minted on re-emit)"
  evidence: internal/core/intent/owed_test.go:101 — "!got["itd-14"].IsOwed()"

## Grounds

- pursued: we expect a count of owed fidelity reviews on the bare status surfaces to be enough to make the debt get paid, because the debt was invisible rather than resisted, and a session that was asked what was outstanding discharged three in one sitting; it is shown wrong if the count is rendered and the debt still accumulates, which would mean visibility was not the constraint
