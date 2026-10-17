---
schema_version: 1
id: "iss-2610170220118643"
slug: "testhandbackscrubleavesnodomain-failed"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "merge-queue failure of the docs pull request #1063"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/cloudscripts/cloud_handback_test.go"
remedy: "Find why the refusal output can arrive without the remote's line (output cut short, or the hook refusing before the remote answers), and make the test wait for or provoke the remote's line so the scrub is always exercised."
---

TestHandBackScrubLeavesNoDomain failed once on the merge queue's macOS race run (Actions run 38991791855, 2026-10-17T02:10Z) for a docs-only pull request: the comment's 'Why the push failed' section carried the pre-push hook's refusal but no 'remote:' line, so the test stopped at cloud_handback_test.go:754 before checking the scrub. The same package's TestHandBackPushesToWIPBranchThroughTheHook has a recorded load-related miss (iss-2610161458469092); this is a second hand-back test whose push output arrives incomplete.
