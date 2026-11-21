---
schema_version: 1
id: "iss-2611211711489111"
slug: "the-away-queue-only-appends-on-21-nov"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "lab-261121153107-bb30bb5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/mode_away.go"
remedy: "Give the away queue a way to close an entry: withdraw a step by its number, and mark one superseded by a later step, each with the check that shows it moot, so the person's return lists only the live asks"
---

The away queue only appends. On 21 Nov 2026 a session queued 21 steps for the person during one away spell, and when the person returned, about half no longer asked for anything: a later step superseded them, the failure they reported had cleared, or the work had already merged. The session had no way to withdraw or supersede an entry, so the person met every stale ask. Two labs (lab-261121150749-bb30bb5, F-5, and lab-261121153107-bb30bb5, F-4) found the same gap.
