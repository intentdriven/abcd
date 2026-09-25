---
schema_version: 1
id: "iss-2609251827296447"
slug: "the-launch-gate-suite-s-proselines-internal-core-launch"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
---

The launch gate suite's proseLines (internal/core/launch/gates.go) reads a Markdown document whose first line is '---' as opening YAML frontmatter and drops every line until a second '---'; a document that opens with a horizontal rule and carries no second rule is dropped whole from both the marker-block and the change-narration gates, silently, in a fail-closed gate. Remedy: frontmatter is skipped only when it closes; an unclosed opening rule is read as prose.
