---
schema_version: 1
id: "iss-2611030450356845"
slug: "the-build-s-land-stage-ships-an-intent"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the land stage's records commit resolve (or ask about) a shipped intent's related issues, or run issue-drift on the records commit before pushing, so the landing the loop makes passes the preflight it requires."
---

The build's land stage ships an intent but leaves the issues the intent names in related_issues open, so the lane's own preflight then fails issue-drift (shipped_unresolved), and the loop refuses any commit after its records commit: on run-2611030127507347 (the memory floor, 2026-11-03) the lane could only be landed outside the loop, with a hand commit resolving iss-2611011737317454.
