---
schema_version: 1
id: "iss-2611032348389895"
slug: "implement-status-without-run-renews-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Make the read form of implement status write nothing, with renewal left to step or an explicit flag; a test runs status against a stand-in remote and asserts no push."
---

implement status without --run renews the reservations of every run in the checkout, which pushes to origin from a verb a reader takes as read-only, and it can start a window as a side effect: a session reading a run's state on 2026-11-03 added record lines and a push it did not expect.
