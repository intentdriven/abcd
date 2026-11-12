---
schema_version: 1
id: "iss-2611121000338677"
slug: "the-layout-statements-that-make-a-record"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "review of the spec readiness change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/specunsettled.go"
remedy: "Add a warn-level record-lint rule over planned intents that reports each layout statement through spec.LayoutStatements, naming the file and line, with a test over a planned intent fixture carrying a four-column fence under a list item."
---

The layout statements that make a record read not ready (a fence indented four columns or more under a list item among them) are reported by record-lint's spec_unsettled rule only for open specs. A planned intent that gains such a layout is found not ready at the start of a build run, through the open_questions check, and not where it is written: nothing reports it at the change that adds it. Found in review of the spec readiness change.
