---
schema_version: 1
id: "iss-2610241212419547"
slug: "testhostversionwarning-failed-once-on-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "setup redesign build, identity lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "make the test's timing budget injectable, or assert behaviour rather than elapsed time"
---

TestHostVersionWarning failed once on a timing check during a loaded test run (load average 20-28) and passed when run alone: a wall-clock assertion that depends on machine load.
