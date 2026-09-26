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
resolution: "The launch page and the brief's launch chapter say binary and application behave identically in every verb."
impact: internal
resolved_by:
  commit: "8740a2d9"
---

The artefact kinds binary and application behave identically in every launch verb (KindBinary is referenced nowhere outside the reader, and the Go leg keys on go.mod), and neither the launch page nor the intent's Decisions says so, so a reader expects the choice to change what the release does.

## Grounds

- pursued: a reader of commands/launch.md learns the two non-plugin kinds are interchangeable; a verb that branches on KindBinary versus KindApplication would make the sentence false and show it wrong
