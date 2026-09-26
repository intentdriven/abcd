---
schema_version: 1
id: "iss-2609251711096816"
slug: "commands-capture-md-s-where-an-rdi-n-comes-from-paragraph"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/capture.md"
resolution: "the plugin page names the shipped ingest verb and the four sub-verbs that act on reading items"
impact: internal
resolved_by:
  commit: "94b775c9f28615aae5e24589ca8e29e0f69313be"
---

commands/capture.md's 'Where an rdi-N comes from' paragraph says the cold-reading ingest verb has not landed and that the reading-item sub-verbs have nothing to act on until it does; the ingest verb ships (abcd reading ingest, commands/reading.md), and four capture sub-verbs now act on reading items (disposition, admit, surprise, promote), so the paragraph states a future that is already the past and counts two verbs.

## Grounds

- pursued: the page now describes the surface as it ships; a reader following it to a verb that does not exist, or missing admit and surprise, would show it wrong
