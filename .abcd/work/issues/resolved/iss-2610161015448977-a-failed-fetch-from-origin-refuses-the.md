---
schema_version: 1
id: "iss-2610161015448977"
slug: "a-failed-fetch-from-origin-refuses-the"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Treat an origin fetch failure as an outage probe and wait as for other lost connections"
related_intents: [itd-2610161016401115]
resolution: "A drain move that cannot fetch origin, cannot read its default branch, or finds no origin/HEAD plans nothing, records an origin probe in the drain's state and waits on the outage schedule instead of refusing"
impact: fix
resolved_by:
  intent: "itd-2610161016394967"
  spec: "spc-2610161044461448"
---

A failed fetch from origin refuses the drain move instead of entering the drain's outage wait.

## Grounds

- pursued: a lost connection to origin makes the drain wait and resume, not stop; shown wrong if a failed fetch refuses a drain move
