---
schema_version: 1
id: "iss-2609091433422134"
slug: "rs001-misattributes-a-queue-collision-to-pre-divergence-history"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
resolution: "RS001 and RS005 recognise a base that is an ancestor of the head, the merge-queue shape, and name the base-side commit that made the record terminal after the branch diverged as a competitor's landing, with rebase-and-reconcile as the remedy; a record terminal before the branch was cut keeps the drop-the-trailer message. Queue-shaped fixture cases in scripts/check-issue-resolution-cases.sh cover both, for issues and intents."
impact: internal
resolved_by:
  commit: "17bf7424c"
---

RS001 refuses correctly on a merge-queue collision and explains it wrongly. When a competitor lands a record's resolution while this entry waits in the queue, the record is already terminal at the entry's base, so it never enters a terminal folder across the range and the trailer is refused, which is the right verdict. The message the refusal carries is the stale-branch one, saying the record already sat in the resolved folder before this branch diverged and advising that the trailer be dropped. The probe behind that wording walks the range from the head back to the base, and in the queue the base is always an ancestor of the head, so that walk is empty and the stale-branch arm is the only one selectable there; the competitor's landing is therefore reported as pre-divergence history that never happened. The cost is a wrong remedy at the worst moment: an author told the record was terminal before they branched will drop a trailer that is correct and re-push, where the real answer is that someone else resolved it while they queued and the two changes need reconciling. The gate only became reachable in the queue when the range gates started resolving a base there, so the arm has never been exercised on this shape. Fix: give the probe a branch that recognises a base that is an ancestor of the head and name the competitor's landing, so the message distinguishes a record that was already terminal when the branch was cut from one that became terminal while it waited. Detector: a queue-shaped range whose record went terminal after the branch point is refused with a message naming the landing rather than the divergence, and a genuinely pre-divergence record keeps the message it has.

## Grounds

- pursued: a merge-queue entry refused because a competitor resolved the same record while it waited is told so, and a pre-divergence resolution in the queue is still told to drop the trailer; a queue refusal that still calls a competitor's landing pre-divergence history would show it wrong
