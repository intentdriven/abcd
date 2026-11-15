---
schema_version: 1
id: "iss-2611150658164454"
slug: "the-inbox-reads-and-lists-every-waiting"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report"
remedy: "Decide a bound for one listing and one run, show the count beyond it, and have the reporting command refuse when the sender already has that many waiting."
---

The inbox reads and lists every waiting report with no bound on how many there are, so one listing or one sweep run can grow without limit.
