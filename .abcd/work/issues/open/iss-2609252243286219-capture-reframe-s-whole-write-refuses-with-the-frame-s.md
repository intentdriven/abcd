---
schema_version: 1
id: "iss-2609252243286219"
slug: "capture-reframe-s-whole-write-refuses-with-the-frame-s"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reframe.go"
---

capture reframe's whole write refuses with 'the frame's previous state cannot be fingerprinted: carries no H2 section titled Construal' when no distinct prior state lies within the history that carries a Construal section, instead of naming the true reason: no prior distinct state within the fingerprintable history, and how far back that history reaches
