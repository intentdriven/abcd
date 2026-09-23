---
schema_version: 1
id: "iss-124"
slug: "foreign-repo-review-receipts-no-home"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "2026-07-25 three-document SOTA/adversarial review"
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M1: plan next cycle as its own intent (a receipt home for foreign-repo reviews and a receipt-backed PR-comment adapter), not folded into itd-28 or itd-83)."
---

Review receipts and the review bar have no home or path for repos-we-don't-own: itd-83 fires reviewer agents only in managed repos and itd-28's receipt store is in-tree only, so outbound PRs to foreign repos carry unfalsifiable prose self-attestations ('Reviewed adversarially; no exploitable path found') instead of receipt-backed claims — the evaluator-inside-the-loop shape itd-58's A4 exploit gate refuses. Live specimen: the maintainer's PR #868 to a third-party repo (2026-07-21, reviewed 2026-07-25 with SOTA + adversarial passes). iss-89 is the routing precedent (foreign-repo work products need a home outside the cwd repo). Two seeds ride on this capture rather than as fresh intents: (1) a same-act deferred-hardenings discipline — any hardening a change consciously defers must exist as a filed, cited issue before the change is presented (generalises workaround-records-the-defect beyond abcd's own defects; mechanically lintable: a PR-body 'Deferred' line without an issue id is detectable); (2) a receipt-backed PR-comment adapter for outbound contributions — receipt lands at home per itd-28, a Stage-1-sanitised rendered summary posts to the forge; SOTA review found the receipt-plus-comment position unoccupied (verdict-as-comment is saturated, SLSA v1.2 leaves review attestations explicitly undefined).