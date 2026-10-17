---
schema_version: 1
id: "iss-2610161015381750"
slug: "lanes-parked-at-land-are-renewed-on"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain_finish.go"
remedy: "On each move read the parked lanes' pull requests in one gh pr list call; stop renewing closed or conflicting ones and name them in next"
related_intents: [itd-2610161016391855]
resolution: "each drain move reads every parked lane from one listing of the person's pull requests: a closed one hands its lane back as closed and releases its reservation, a conflicting one is no longer renewed and is named, an open one is renewed and named as waiting"
impact: fix
resolved_by:
  commit: "40b14c21ba2e5d03504a3678152872ab8ba8f3bd"
---

Lanes parked at land are renewed on every drain move from local state alone, whatever their pull request's state, and a lane whose head is not on main is skipped silently, so a pull request closed, dropped from the merge queue or conflicting after arming keeps its issue reserved and counted against the cap indefinitely and next never names it.
