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

<!-- abcd-review: OWED receipt=rcp-dd80727d7bd1 -->
Fidelity review OWED (receipt rcp-dd80727d7bd1).

## Grounds

- pursued: we expect a count of owed fidelity reviews on the bare status surfaces to be enough to make the debt get paid, because the debt was invisible rather than resisted, and a session that was asked what was outstanding discharged three in one sitting; it is shown wrong if the count is rendered and the debt still accumulates, which would mean visibility was not the constraint
