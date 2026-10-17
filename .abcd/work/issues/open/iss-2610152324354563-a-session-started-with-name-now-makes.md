---
schema_version: 1
id: "iss-2610152324354563"
slug: "a-session-started-with-name-now-makes"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "fix round 4 of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Resolve the root lazily, only on the paths that read the persona list; test that a titled startup runs no git."
refines: [iss-2609091729516940]
---

A session started with --name now makes the session-start hook resolve the repository root (a git call) on its keep path, which only records an empty given entry and needs no repository.
