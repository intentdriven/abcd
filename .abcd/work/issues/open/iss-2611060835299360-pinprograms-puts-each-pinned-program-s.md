---
schema_version: 1
id: "iss-2611060835299360"
slug: "pinprograms-puts-each-pinned-program-s"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Name the trade-off in the wall's notes and at a stage's start, or pin by program rather than by folder where that keeps a pinned name's meaning."
---

PinPrograms puts each pinned program's whole folder ahead of everything else on PATH (`internal/core/runner/wall.go:58`), so a repository whose person has a python3 or node version manager first on PATH gets a package manager's or the system's copy in a walled stage. The stage's tools then differ from the person's terminal with nothing naming it, and a build or test can pass or fail on the wrong version.
