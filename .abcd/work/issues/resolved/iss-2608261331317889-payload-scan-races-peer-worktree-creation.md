---
schema_version: 1
id: "iss-2608261331317889"
slug: "payload-scan-races-peer-worktree-creation"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "pre-push-preflight"
found_at: "internal/core/launch"
resolution: "decided: adr-2609091248200336 moves session worktrees out of the checkout, and AGENTS.md names this flake signature a retry during worktree churn"
impact: internal
---

TestPayloadTreeImplementationsResolveIdentically failed transiently during a pre-push preflight with 'the payload carries 109 rejected file(s)', then passed in isolation minutes later with an identical two-markdown-file diff and a clean tree. Between the two runs a peer session created .claude/worktrees/pr513-conflicts in the shared checkout, so the plausible mechanism is the payload walk sweeping a sibling worktree mid-population — the iss-213 class (a peer's activity silently invalidating a verification result), now reaching the payload scan through the preflight gate. Not reproduced deterministically; captured as a sighting so the flake has a marker if it recurs.

## Grounds

- pursued: with worktrees kept out of the checkout, the payload walk no longer sweeps a peer's half-populated tree; shown wrong if the payload flake recurs with no worktree being created inside the checkout
