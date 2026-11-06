---
schema_version: 1
id: "iss-2611061151024373"
slug: "reflect-write-counts-an-intent-with"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "writing the v0.16.2 retrospective"
origin: researcher-authored
production_mode: hand-written
found_at: "conceptual: the reflect verb's unshipped-targets reading"
remedy: "Have `reflect` and `reflect write` count an intent targeting next only for a release not yet tagged, and name it as carried over rather than unshipped."
---

`reflect write` counts an intent with `target_release: next` as owed by a release already cut: writing the v0.16.2 retrospective after its tag was refused over itd-2611041549461012, which targets next, not that tag. So the reflection asks the person to confirm unshipped work that was never promised to the release.
