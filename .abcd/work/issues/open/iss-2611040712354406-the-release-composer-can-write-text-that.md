---
schema_version: 1
id: "iss-2611040712354406"
slug: "the-release-composer-can-write-text-that"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ship.go"
remedy: "Have launch ship's practice run assemble each reading position over the tree it would write, and refuse where one refuses."
---

The release composer can write text that breaks a later verb: the v0.16.1 changelog's first draft carried an HTML heading tag in a code span, which reading assemble read as a raw heading to the end of the file and refused at three positions, and neither the ingest nor the docs gate noticed. The reader's bug is captured separately; the gate between the composer and the commit runs no such consumer.
