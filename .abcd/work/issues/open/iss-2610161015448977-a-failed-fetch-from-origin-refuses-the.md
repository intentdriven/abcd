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
---

A failed fetch from origin refuses the drain move instead of entering the drain's outage wait.
