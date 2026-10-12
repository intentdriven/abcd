---
schema_version: 1
id: "iss-2610120138148399"
slug: "the-implement-loop-s-conflict-probe"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ruthless review of the loop's conflict refusal (iss-2610110908443341, PR #960)"
origin: researcher-authored
production_mode: hand-written
found_at: "the implement loop's conflict refusal (conflictRefusal in internal/core/implement/loop/conflict.go, added by PR #960)"
remedy: "probe the pull request's current head (the remote branch tip, or gh's headRefOid) rather than ld.Pushed, with a test that hand-merges one conflict and leaves a second"
---

The implement loop's conflict probe (internal/core/implement/loop/conflict.go, conflictRefusal) predicts the forge's merge from the head the loop pushed, ld.Pushed, not the branch's current tip. The refusal's remedy is a person's hand merge, pushed to the lane's branch; that push does not change ld.Pushed. So if the pull request is still reported CONFLICTING afterwards (a second conflict, or the forge not yet recomputed), the probe re-merges the old head and can name a path the person has already resolved, sending them to fix what is fixed.
