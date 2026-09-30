---
schema_version: 1
id: "iss-124"
slug: "foreign-repo-review-receipts-no-home"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "2026-07-25 three-document SOTA/adversarial review"
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md"
remedy: "Waits on the M1 planning interview: file the foreign-review intent so a review of a repository abcd does not manage writes a receipt wherever the interview puts it (this repository's work tier, the machine-scoped store, or the foreign repository), and the PR-comment adapter posts only a rendered summary of an existing receipt and refuses without one. Prove it with a test that the adapter refuses a post naming no receipt, and a lint that a pull-request body's Deferred line carries an issue id."
deferred_after: v0.11.1
deferral_reason: "The product thinker's ruling M1 of 2026-09-23: planned next cycle as its own intent, not folded into itd-28 or itd-83. Owed: that intent's filing and interview, which opens on one question: do receipts of a review of a foreign repository live in this repository's work tier, the machine-scoped store, or the foreign repository, and does the PR-comment adapter post only from them?"
---

Review receipts and the review bar have no home or path for repos-we-don't-own: itd-83 fires reviewer agents only in managed repos and itd-28's receipt store is in-tree only, so outbound PRs to foreign repos carry unfalsifiable prose self-attestations ('Reviewed adversarially; no exploitable path found') instead of receipt-backed claims — the evaluator-inside-the-loop shape itd-58's A4 exploit gate refuses. Live specimen: the maintainer's PR #868 to a third-party repo (2026-07-21, reviewed 2026-07-25 with SOTA + adversarial passes). iss-89 is the routing precedent (foreign-repo work products need a home outside the cwd repo). Two seeds ride on this capture rather than as fresh intents: (1) a same-act deferred-hardenings discipline — any hardening a change consciously defers must exist as a filed, cited issue before the change is presented (generalises workaround-records-the-defect beyond abcd's own defects; mechanically lintable: a PR-body 'Deferred' line without an issue id is detectable); (2) a receipt-backed PR-comment adapter for outbound contributions — receipt lands at home per itd-28, a Stage-1-sanitised rendered summary posts to the forge; SOTA review found the receipt-plus-comment position unoccupied (verdict-as-comment is saturated, SLSA v1.2 leaves review attestations explicitly undefined).

## Deferral 2026-09-29

Deferred past v0.11.1: The product thinker's ruling M1 of 2026-09-23: planned next cycle as its own intent, not folded into itd-28 or itd-83. Owed: that intent's filing and interview, which opens on one question: do receipts of a review of a foreign repository live in this repository's work tier, the machine-scoped store, or the foreign repository, and does the PR-comment adapter post only from them?

## Remedy grounds (2026-09-29)

- The home of the receipt is the interview's one open question, so the remedy fixes everything but the location: the receipt is the evidence, the comment only renders it, which keeps the evaluator outside the loop.
- SOTA check: SLSA v1.2 source track (https://slsa.dev/spec/v1.2/source-requirements, read 2026-09-29) names a code-review attestation only as an example and leaves its format to the source control system, so no standard format exists to adopt; the receipt shape itd-28 already uses stays the format.
- Rejected: a verdict-as-comment bot with no stored receipt, the self-attestation the record refuses.
