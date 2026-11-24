---
schema_version: 1
id: "iss-2611240046353801"
slug: "while-the-person-is-away-the-stop-judge"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/judge.go"
remedy: "Count a behind pull request as owed only where the repository does not merge through a queue, or where the forge refuses it for being out of date; one armed through a merge queue is waiting, not behind."
refines: [iss-2610311339274372]
---

While the person is away, the stop judge tells the session to merge main into a pull request that is behind its base even when that pull request is armed through the merge queue, which tests the merge with main itself; following it restarts the pull request's CI for nothing.
