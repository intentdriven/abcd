---
schema_version: 1
id: "iss-2610161015368630"
slug: "a-lane-that-refuses-at-a-step-without"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "After a lane's step has refused the same way N times or for T minutes, the drain hands it back as stuck, releases its reservation and moves on (grounds: dead-letter redrive, as SQS maxReceiveCount)"
related_intents: [itd-2610161016391855]
---

A lane that refuses at a step without handing back (a conflicting pull request at arming, the land fence, or a brief refused because the issue is already resolved at the base) is read by laneOutcome as in progress, so every later drain move answers "drive" that lane and the drain opens nothing more; implement step --discard and --release act only on held lanes, so only removing the run directory by hand frees it.
