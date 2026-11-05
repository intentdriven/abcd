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
remedy: "abcd land withdraws the arming before a push that moves an armed pull request's head, and arms the new head only where the merge gate does; the committed pre-push hook refuses a fast-forward it reads as moving that head"
refines: [iss-2611011943357182, iss-2608241347321757, iss-2610110012045504]
resolution: "abcd land withdraws the arming before a push that moves the head, and the committed pre-push hook refuses a fast-forward it reads as moving it"
impact: fix
resolved_by:
  commit: "49de3c1a656c"
---

abcd land withdraws the arming before a push that moves an armed pull request's head, and arms the new head only where the merge gate does; the committed pre-push hook refuses a fast-forward it reads as moving that head.

## Grounds

- pursued: abcd land keeps an arming to a head the merge gate arms; shown wrong if abcd land pushes over an arming it read and leaves that arming on
