---
schema_version: 1
id: "iss-2610142326511964"
slug: "macos-merge-queue-runs-slowed-to-34-40"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "raising the macOS check job limit to 60 minutes"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Profile the macOS race lane (the one race-enabled whole-module pass) and split or cache it before merge-queue runs reach the 60-minute job limit."
---

The macOS check job's merge-queue runs grew from 31-35 minutes to 34-40 minutes on 2026-10-14 (#990 took 40.0 minutes), close enough to the 45-minute job limit that the limit was raised to 60. The extra time is unexplained, and if it keeps growing the new limit will be reached too.
