---
schema_version: 1
id: "iss-2611100548341325"
slug: "some-hook-git-calls-still-run-with-no"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "lane p fix round 3 of spc-2610230805429429"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/view.go"
remedy: "Thread a context through claim's view (localGit, storedGit, the tip batch, originDefault, generatedFilesAt) and peers.RecordsAt, and through landing.Receipt (gitutil.ListWorktrees and localtier.Untracked), as gitutil's Context forms do, bind each listed hook's git calls to that hook's deadline, add a gate that refuses a line added to internal/surface/cli/testdata/hook-git-baseline.txt against the merge base (the audit, TestHookGitCallsMatchBaseline, refuses new and rising calls but not a hand-added line), then, once that audit's baseline is empty, restore the universal claim in the checklist pages, the stop check's comment and dcn-2611100548371812"
---

Some hook git calls still run with no deadline. The stop check's owed read binds the session's identity and holder, the default branch and the last-fetch stamp, but not every read. The reservations' view (claim.View and the peers.RecordsAt it reaches) reads refs, tips, the default branch's records and generated files through about 37 call sites shared with the reservation verbs. The landing-receipt read (hasReceipt in internal/core/autonomy/owedread.go, reached twice in the owed read) calls landing.Receipt, which lists worktrees through gitutil.ListWorktrees and checks the receipt with localtier.Untracked, both with no deadline. Other unbound callees may remain; nobody has traced the whole read. Outside the stop check, the session-start away block (managedRoot, autonomy.Resolve, NoteSessionCopy), the PreCompact record (recordCompaction), the StopFailure hook, the session-start transcript drain and the guard's branch recording also call git with no deadline, so a git that hangs holds the hook until the host's own timeout.
