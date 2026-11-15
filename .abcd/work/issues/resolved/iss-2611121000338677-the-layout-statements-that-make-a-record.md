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
resolution: "record-lint's intent_layout rule, armed at warn, names each layout statement spec.LayoutStatements finds in a draft or planned intent (a fence or HTML comment that never closes, or fences or comments whose extent turns on the layout around them, a fence indented four columns under a list item among them) on its line, with the remedy the build's open_questions row names, read from the text the intent-tree scan already read. Drafts are read because that row judges them too. On this repository it reports nothing today; the two planned intents that read not ready are held by open questions in their specs (spec_body), not by layout."
impact: fix
resolved_by:
  commit: "78398a6395d7c508d21cd6fa1a5a3d429792e467"
---

The layout statements that make a record read not ready (a fence indented four columns or more under a list item among them) are reported by record-lint's spec_unsettled rule only for open specs. A planned intent that gains such a layout is found not ready at the start of a build run, through the open_questions check, and not where it is written: nothing reports it at the change that adds it. Found in review of the spec readiness change.
