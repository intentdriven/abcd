---
schema_version: 1
id: "iss-2608290822140563"
slug: "the-fidelity-audit-runs-after-the-merge-so-its-verdict-arriv"
severity: "major"
category: "process"
source: "user-observation"
found_during: "intent-implementation-run"
found_at: "internal/core/intent/audit.go"
---

The fidelity audit runs after the merge, so its verdict arrives when the cheap remedies are already gone, and the code refuses any other ordering: the ingest rejects an intent that is not in the shipped bucket, and the emit fires from the ship move itself, so an intent cannot be audited against the candidate diff on a branch. The stated reason is sound as far as it goes, that a report-only review must never un-ship what has already shipped, but it answers a question that only arises because the audit was placed after the merge in the first place. Audited before the merge, a failing criterion has three cheap answers: fix the branch, revise the promise before making it, or decline to merge. Audited after, it has none, because there is no un-ship path and the code is on the trunk. The counter-argument is real and should not be waved away: a branch is not the tree that lands, since the merge queue lands merge commits and a semantic conflict with a concurrently merging change can alter the delivered reality after the verdict was formed, so the post-merge audit judges what is actually true while a pre-merge one judges a candidate. The shape that gets both is to gate on the pre-merge verdict and bind it to a content hash of the tree it judged, then check deterministically after the merge that the landed content still matches, reopening the receipt only on a mismatch, which costs one hash comparison rather than a second audit and reuses the content-addressing the transcript store already relies on. Whatever the ordering, the model's verdict must stay a proposal and a human acknowledgement must remain the gate, because a non-deterministic verdict that can block a merge on its own is a trust step this repository has not taken and would train people to route around the gate.

Reframed 2026-08-29 under adr-55: the agents run autonomously and stop only to obtain a verdict, which decides this. The audit IS the stop, so it belongs before the merge, where a failed criterion still has cheap answers. A post-merge audit never stops the loop; it leaves a note. The content-hash check after the merge remains the safety net for the branch-is-not-the-landed-tree objection.
