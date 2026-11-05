---
schema_version: 1
id: "iss-2611050535437225"
slug: "a-push-that-would-move-an-armed-pull"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "a landing fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/withdraw.go"
remedy: "abcd land withdraws the arming before its push and arms the new head once the merge gate does; the committed pre-push hook refuses a push that moves an armed pull request's head"
refines: [iss-2611011943357182, iss-2608241347321757, iss-2610110012045504]
resolution: "abcd land withdraws the arming before a push that moves the head, and the pre-push hook refuses such a push"
impact: fix
resolved_by:
  commit: "49de3c1a656c"
---

A push that would move an armed pull request past its reviewed head is refused until it is re-reviewed.

## Grounds

- pursued: an arming stays on the head a review named; shown wrong if a push that moves an armed head goes ahead through abcd land or the committed hook with the arming still on
