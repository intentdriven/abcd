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
remedy: "Thread a context through claim's view (localGit, storedGit, the tip batch, originDefault, generatedFilesAt) and peers.RecordsAt, as gitutil's Context forms do, bind each listed hook's git calls to that hook's deadline, then drop the named exception from the checklist pages and the stop check's comment"
---

Some hook git calls still run with no deadline. The stop check's owed read binds every git call of its own except the reservations' view (claim.View and the peers.RecordsAt it reaches), which reads refs, tips, the default branch's records and generated files through about 37 call sites shared with the reservation verbs. Outside the stop check, the session-start away block (managedRoot, autonomy.Resolve, NoteSessionCopy), the PreCompact record (recordCompaction), the StopFailure hook, the session-start transcript drain and the guard's branch recording also call git with no deadline, so a git that hangs holds the hook until the host's own timeout.
