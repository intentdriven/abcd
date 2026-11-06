---
schema_version: 1
id: "iss-2611061902568865"
slug: "release-binary-told-to-rebuild-from"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first real drain ahead of a long run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Say this only for a build that is not a release; for a release, name the version and leave it there, and never append it to a usage refusal."
---

A released abcd binary run in a checkout of its own repository tells the person it is behind its own source and to rebuild with make build, naming the commit it was built from. For someone deliberately on the latest release that advice is wrong, and following it replaces the release they chose to run. The same line follows an unrelated usage refusal, so it reads as the cause of the refusal.
