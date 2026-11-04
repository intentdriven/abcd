---
schema_version: 1
id: "iss-2611040554340976"
slug: "the-land-chapter-says-the-json-carries"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Set Pushed only when the porcelain result shows the ref moved, or restate the field in the chapter as the head a successful push left on origin."
---

The land chapter says the JSON carries `pushed` when the push sent a commit (.abcd/development/brief/04-surfaces/38-land.md:299). push() sets res.Pushed = res.Head on every successful push before it reads the porcelain result (internal/core/landing/land.go:882), so a no-op push whose step note says origin already held the branch and the push changed nothing (land.go:885) carries `pushed` as well.
