---
schema_version: 1
id: "iss-2611101318528163"
slug: "the-build-loop-s-pull-request-title"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Compose the title from the step's plain text: strip markdown emphasis and code marks and drop a trailing parenthetical note; a test titles a step whose text holds bold markers and finds none in the title."
---

The build loop's pull-request title copies the spec step's title as written, markdown included. prTitle (internal/core/implement/loop/land.go) puts the lane's step title into the title unchanged, so on 7 Nov at 21:35 the title of #1451 carried bold markers and a trailing parenthetical note from the spec step, and a person had to rewrite it. See also the different fault in the same composer, iss-2610080546203039: a title that names a record id with no Refs: line for it.
