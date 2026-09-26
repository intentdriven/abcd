---
schema_version: 1
id: "iss-2609260149248713"
slug: "the-artefact-kinds-binary-and-application-behave-identically"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/kind.go"
---

The artefact kinds binary and application behave identically in every launch verb (KindBinary is referenced nowhere outside the reader, and the Go leg keys on go.mod), and neither the launch page nor the intent's Decisions says so, so a reader expects the choice to change what the release does.
