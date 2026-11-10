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
remedy: "Bind the guard hook's reservations' view, reached through sweep's delete prover (claim.View in deleteProver.reservationsRead and deleteProver.held), to that hook's deadline with claim.ViewContext, the form the stop check's reservations read now uses, which lowers the ten baseline entries on the view's route that the guard still reaches; bind claim's readers that keep state or are reached on other hook routes (Fetch and the kept first sighting, Finished, RecordOnDefault, defaultRecords, siblingCopies and NameSiblings); bind the receipt finders' other hook routes (findReceipt and findReceipts, reached through the review and approval receipts) and each listed hook's other git calls to that hook's deadline, add a gate that refuses a line added to internal/surface/cli/testdata/hook-git-baseline.txt against the merge base (the audit, TestHookGitCallsMatchBaseline, refuses new and rising calls but not a hand-added line), then, once that audit's baseline is empty, restore the universal claim in the checklist pages, the stop check's comment and dcn-2611100548371812"
---

Some hook git calls still run with no deadline. The stop check's owed read binds the session's identity and holder, the default branch, the last-fetch stamp and the landing-receipt read, but not every read. The reservations' view (claim.View and the peers.RecordsAt it reaches) reads refs, tips, the default branch's records and generated files through about 37 call sites shared with the reservation verbs. The stop check's receipt read (hasReceipt in internal/core/autonomy/owedread.go) is bound through landing.ReceiptContext, but the deadline-free finders it sits beside, findReceipt and findReceipts in internal/core/landing/primitives.go, are still reached from a hook on the review and approval receipt routes, and list worktrees through gitutil.ListWorktrees and check each receipt with localtier.Untracked with no deadline. Other unbound callees may remain; nobody has traced the whole read. Outside the stop check, the session-start away block (managedRoot, autonomy.Resolve, NoteSessionCopy), the PreCompact record (recordCompaction), the StopFailure hook, the session-start transcript drain and the guard's branch recording also call git with no deadline, so a git that hangs holds the hook until the host's own timeout.
