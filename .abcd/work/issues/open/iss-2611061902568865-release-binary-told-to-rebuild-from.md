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
found_at: "internal/surface/cli/staleusage.go"
remedy: "Say this only for a build that is not a release; for a release, name the version and leave it there, and never append it to a usage refusal. internal/surface/cli/staleness.go prints the same advice at session start and needs the same change and test. internal/core/ahoy/vintage.go gives it too: there only the advice changes, and its refusal of an install from a binary behind its source stands."
---

A released abcd binary run in a checkout of its own repository tells the person it is behind its own source and to rebuild with make build, naming the commit it was built from. For someone deliberately on the latest release that advice is wrong, and following it does nothing for them: make build writes a build under bin/ in the checkout, so the copy that gave the advice is the one that still runs. The same line follows an unrelated usage refusal, so it reads as the cause of the refusal.
