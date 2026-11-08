---
schema_version: 1
id: "iss-2611080058096268"
slug: "abcd-implement-status-shows-a-run-s"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "intent audit of itd-2611041549461012, run-2611071001420640"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Show the live driver (session, process, start), the standing launches and each lane's uncommitted count on abcd implement status, from the same records the resume listing reads."
---

abcd implement status shows a run's driver only by its kind. The live driver, each runner launch and each lane's uncommitted work are recorded, but only abcd implement resume lists them, so a person who checks status cannot see who drives the run or what a lane left uncommitted.
