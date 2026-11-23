---
schema_version: 1
id: "iss-2611232350047170"
slug: "while-the-person-is-away-the-stop-judge"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
remedy: "Treat a whole run as live for a failing or behind pull request when the run's head is a local descendant of the pull request's head on the same branch, so the wake releases the stop while the fix is being checked."
---

While the person is away, the stop judge counts a pull request with failing checks as stuck even when a preflight is already running on the commit that fixes it, because it treats the run as live only for the head the pull request already has. The session is told to move an item it cannot move until that preflight ends.
