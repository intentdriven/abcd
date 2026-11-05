---
schema_version: 1
id: "iss-2611051719447823"
slug: "a-first-signal-during-a-drive-s-login"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
remedy: "Check for a stop between the login check and the launch."
---

A first signal during the login check of `abcd drain --drive` still starts the proof's launch: the signal marks a soft stop, and nothing reads it between the check and the launch. The drive is not on the default branch yet, so this record carries no location.
