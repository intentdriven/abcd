---
schema_version: 1
id: "iss-2610150524246898"
slug: "enqueue-every-ready-pull-request-at-once"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "v0.15.0 run review 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "When a pull request abcd lands has its receipt and is pushed, put it straight into the repository's merge queue (or arm auto-merge where that is the forge's route), never waiting for an earlier entry to merge, so the queue builds the entries its own settings allow at once. Toggle with a config key, default on. Never change the repository's queue settings: where no queue gates the default branch, say so and fall back to auto-merge or leave the merge to the person."
---

Ready pull requests reach the merge queue one at a time, so a queue that can build four entries at once tests one. On 2026-10-15 abcd's own queue was configured to build up to four entries (max_entries_to_build 4), but the coordinator queued each pull request only after the previous one merged, through hand-run watchers. With each queue run taking 25 to 40 minutes, three ready pull requests cost about two hours instead of about forty minutes. abcd's land step arms auto-merge for a build lane, which lets the queue batch, but landings outside the loop have no such step, and iss-2610090642376032 already records that even a lane's arming can leave a pull request unqueued. The product thinker asked (2026-10-15) for queueing ready work together to be the default for every abcd-managed repository, toggled in config. Bound by the security standard rule: abcd adds the queueing, never changes how the repository's queue is configured.
