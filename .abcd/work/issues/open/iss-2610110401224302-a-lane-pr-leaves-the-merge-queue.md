---
schema_version: 1
id: "iss-2610110401224302"
slug: "a-lane-pr-leaves-the-merge-queue"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "While a lane waits at land for its merge, have implement step read the pull request's auto-merge and merge-queue state through gh. When it was armed and is no longer armed or queued, say so in the refusal, naming the removal and the merge group's conclusion. If the merge group was cancelled or superseded, re-arm once and record the re-arm in the run record. If the merge group failed, return the lane to a fix round briefed on a merge of the default branch and the failing check's output, rather than re-arming or reporting only 'not merged yet'."
---

A lane's pull request can leave the merge queue without anyone being told, and its auto-merge falls off with it. On 2026-10-11 the queue removed two drain PRs. #906's queue CI was cancelled when a peer's PR was re-queued ahead of it. #903 was removed about 45 minutes after entry while its macOS check was still running. Both then sat open and unarmed. Meanwhile implement step went on reporting only 'pull request #N is not merged yet', because it reads merge state, never queue membership or the armed flag. They were found by hand an hour or more later.

A second occurrence, on 2026-11-01, shows that re-arming once is not always the answer. #1315 (lane 1 of run-2610312053481114) was armed at 06:13 and queued behind #1313. The queue ran the two together and `go vet` failed: each branch had added a test helper named `treeHashes` to the same package, so the merge group declared it twice. A comment on the pull request explained the clash, but the queue removed it at 07:29 and it sat open, green and unarmed for over four hours, until the build session looked. Merging main also changed what one of its tests asserted: #1314, landed meanwhile, raised the default rows limit from 24 to 32. Each branch passed on its own, and only the combination failed. A re-arm here would have failed the same way. So whatever reads the queue's removal should also read the merge group's conclusion. A failed merge group should send the lane back to a fix round on a merge of the default branch, rather than re-arming.
