---
schema_version: 1
id: "iss-2611032348394774"
slug: "the-roles-live-in-the-account-wide"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/config.go"
remedy: "Allow a role override per run or per session, recorded in the run state, so a trial changes nothing for its neighbours; a test sets an override on one run and asserts another run keeps the account's route."
---

The roles live in the account-wide config.json, so routing one test's roles to the host reroutes every stage any session in that account starts: on 2026-11-03 a walled test's backup-and-edit also changed where the release session's stages would go, and a test stopped before any stage has no rule for putting it back.
