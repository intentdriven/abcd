---
schema_version: 1
id: "iss-186"
slug: "capture-transition-remove-failure"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bug-hunt loop round 1"
found_at: "internal/core/capture/workflow.go:266"
resolution: "commitTransition now rolls back the destination on a non-ENOENT remove failure instead of leaving the issue split across open/ and resolved/ (internal/core/capture/workflow.go); regression test TestTransitionRemoveFailureDoesNotStrandIssueInTwoDirs watched failing before, passing after."
impact: fix
---

capture.commitTransition writes the destination file before removing the source (workflow.go:255-273); a non-ENOENT os.Remove(src) failure (EPERM/EROFS/EIO — e.g. a read-only remount or restrictive ACL on the source status dir) returns an error after dst already landed, with no rollback of dst. The same issue id is now present in two status directories (e.g. both open/ and resolved/). findIssue (alloc.go:383) then rejects any further transition on that id as ErrDuplicateIssueID, so it can never again be resolved or wontfixed on a healthy filesystem — no repair verb exists; recovery requires manually deleting one copy. scanLedger-based reads (List/Status) do not dedupe either, so the id also double-counts. Confirmed with a real non-ENOENT unlink failure (ext4 immutable attribute on the source dir, which blocks unlink even as root, unlike chmod). Reproducing test: internal/core/capture/workflow_test.go, TestTransitionRemoveFailureDoesNotStrandIssueInTwoDirs.