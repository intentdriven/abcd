---
schema_version: 1
id: "iss-288"
slug: "the-adversarial-review-principle-has-no-armed-detector-adver"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
found_at: ".abcd/development/principles/adversarial-review-scales-with-blast-radius.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Should intent ready report, and intent plan refuse without, adversarial-review receipts?"
remedy: "Waits on ruling (should intent ready report, and intent plan refuse without, review receipts): if yes, define a review receipt on the draft (reviewer lens, date, verdict, findings applied or rejected, the draft's content hash) in the .abcd/work/reviews/ shape, have intent ready report the count as a check and intent plan refuse a draft with fewer than two receipts at its current hash, proven by a plan test refusing one receipt and a stale hash; if no, have intent ready report the count as advice and amend the principle to say the rule is self-enforced."
---

The adversarial-review principle has no armed detector: adversarial-review-scales-with-blast-radius requires two independent reviews before an intent moves drafts/ to planned/ (and one before an ADR is accepted), but nothing refuses the move without them. The enforceable rung: intent ready (and the record checks) refuse the transition unless review receipts exist in the .abcd/work/reviews/ VSA shape for the draft at its reviewed content hash
**Corroboration (2026-09-20, Gropius session gropiusllm-97, relayed to
abcd-17).** Met from the managed-repository side at v0.9.0: the planning
interview's prerequisite of two adversarial reviews (design/feasibility and
record-discipline) was carried out on a draft, and the session found no record
shape the readiness gate could count, so the prerequisite is enforced by the
session's own honesty. The intent surface page says as much ("the readiness
gate that will refuse the move mechanically is a recorded seed until built");
this record is that seed. The ask is a receipt the reviews leave on the draft
(reviewer lens, date, verdict, findings applied or rejected) that `intent
ready` reports as a check and `plan` refuses without.

## Remedy grounds (2026-09-29)

Why: the principle already sets two reviews before planned, and a receipt keyed to the content hash is what stops a review of an earlier draft counting for a changed one. Rejected: counting DECISIONS.md lines as reviews, which is the unauditable prose the corroborating session found.
