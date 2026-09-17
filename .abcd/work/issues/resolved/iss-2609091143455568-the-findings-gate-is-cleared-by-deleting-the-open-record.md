---
schema_version: 1
id: "iss-2609091143455568"
slug: "the-findings-gate-is-cleared-by-deleting-the-open-record"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog"
deferred_after: "v0.7.1"
deferral_reason: "The gate's own hardening is where this belongs, and this is the cut that introduced the gate. Fixing it here would mean changing GuardFindings while the same cut is the first to run it, so the change would have no release behind it that had exercised the old behaviour and no evidence of the new one beyond its own tests. Deferred to the cycle after v0.7.1's successor, where the sibling gate's answer is the model: RS001 already refuses a bare delete, so the fix is to ask its question rather than to invent one. Recorded rather than left unsaid because the finding was surfaced by an adversarial review of this very gate, and a gate whose first act is to swallow a finding about itself is the failure the principle behind it names."
resolution: "GuardFindings now also asks whether a blocking record the anchor held in open/ is absent from the whole ledger at HEAD: a deletion refuses under the new release refusal kind deleted-finding, naming the record and the grade the anchor held, while a resolution, a wontfix, a re-slug and a standing deferral each go on clearing the gate. The residual is stated on the gate itself: a record captured after the anchor and deleted before HEAD appears in neither tree, and a git-log walk cannot recover it under squash or rebase merges."
impact: fix
---

GuardFindings decides whether a finding has been answered by asking only whether its id is still under the open bucket at HEAD, so deleting the record clears the gate as effectively as resolving it. Every legitimate route leaves a trace the next reader can follow, a resolution, a wontfix, or a deferral with its reason, and the illegitimate one leaves nothing at all, which makes it both the cheapest way past the gate and the only way that destroys the finding rather than answering it. The issue-resolution gate refuses exactly this shape already, RS001 holding that a bare delete of an open record satisfies no trailer, so the two gates disagree about what counts as an answer while sitting in the same release path. The fix is to ask the question the sibling gate asks: a record present at the anchor and absent at HEAD is a deletion rather than a disposition, and the gate should refuse it and name it, since a record that entered the ledger and left it without reaching a terminal folder is the one case nobody can audit afterwards. Detector: a cut whose only change to a blocking record is its deletion must refuse, naming the record and the deletion, while a resolution, a wontfix and a deferral each continue to clear it. Found by adversarial review of the gate's own release; deferred rather than fixed in that cut because the fix belongs with the gate's own hardening and the release it would have blocked is the one that introduced it.

## Grounds

- pursued: asking the sibling gate's question closes the only route that destroys a finding; it would be shown wrong by a cut that refuses a legitimate disposition, which the disposition table asserts against
