---
schema_version: 1
id: "iss-2611070737171896"
slug: "deferred-by-decision-in-a-drive-the-read"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive.go"
remedy: "Register the read so it runs after the signal handling closes (now that close waits, this is exact), and test a signal at that moment."
---

Deferred by decision: in a drive, the read of whether a stop was asked runs in a defer before signal handling closes, so a signal landing just as a failing try returns is missed: the drive prints 'stopping' and the retry can still launch a live proof. A second signal still ends it.
