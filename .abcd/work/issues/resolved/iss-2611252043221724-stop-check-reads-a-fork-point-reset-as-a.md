---
schema_version: 1
id: "iss-2611252043221724"
slug: "stop-check-reads-a-fork-point-reset-as-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Leave a commit out of a branch's recorded tips when any line of its reflog says the branch was created there, with a test that resets a branch to its fork point."
resolution: "A commit any line of a branch's reflog says the branch was created at no longer counts as one of its tips, so a branch reset to its fork point is still the branch cut from the other. TestForkPointResetIsNoStack failed before the change and passes after it."
impact: fix
---

While the person is away, the stop check says that a branch is stacked on one that was cut from it, and asks for a merge the wrong way round. A branch reset to the commit it was created at carries that commit in its reflog under a reset line as well as under its creation line; stackIdx skips the creation line alone, so the fork point counts as a tip of the later branch, and the branch it was cut from reads as stacked on it. Met on 2026-11-25 after two commits were made again with a soft reset, and shown with a scratch repository.
