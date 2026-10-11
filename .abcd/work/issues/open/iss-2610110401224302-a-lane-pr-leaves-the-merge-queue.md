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
remedy: "While a lane waits at land for its merge, have implement step read the pull request's auto-merge and merge-queue state through gh; when it was armed and is no longer armed or queued, say so in the refusal (removed from the queue, with the queue run's conclusion) and re-arm once, recording the re-arm in the run record, rather than reporting only 'not merged yet'."
---

A lane's pull request can leave the merge queue without anyone being told, and its auto-merge falls off with it. On 2026-10-11 the queue removed two drain PRs. #906's queue CI was cancelled when a peer's PR was re-queued ahead of it. #903 was removed about 45 minutes after entry while its macOS check was still running. Both then sat open and unarmed. Meanwhile implement step went on reporting only 'pull request #N is not merged yet', because it reads merge state, never queue membership or the armed flag. They were found by hand an hour or more later.
