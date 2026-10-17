---
schema_version: 1
id: "iss-2610161458469092"
slug: "testhandbackpushestowipbranchthroughtheh"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "merge-queue failure of the cloud-sessions pull request"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/cloudscripts/cloud_handback_test.go"
remedy: "On the next recurrence, read the printed hand-back output, find what refused or dropped the push under load, and fix it at that cause; resolve this record if it does not recur in the next ten CI runs of the package."
resolution: "The pre-push hook piped git archive into tar to read the cloud connector declarations; a tar stopping at the end-of-archive marker left git archive writing into a closed pipe, and under pipefail the read failed and the hook refused the work-in-progress push, naming no cause. The hook now reads an archive file and reports what either command said. TestHandBackHookReadsTheConnectorsFromAFile reproduces the refusal with a tar that stops reading early and fails on the old hook."
impact: internal
resolved_by:
  commit: "ccc801897"
---

TestHandBackPushesToWIPBranchThroughTheHook missed its first push twice in three runs while the machine's load was about 128 just after a reboot: origin held nothing on the work-in-progress branch. The test printed none of hand-back's output then, so the cause is unknown; 31 later runs at load 20 to 50 passed. The test now prints hand-back's output when that push misses, so a recurrence shows its cause.

## Grounds

- pursued: a work-in-progress push no longer fails on how tar reads its input; a recurrence would now print the read's own error in the refusal, which would show this wrong
