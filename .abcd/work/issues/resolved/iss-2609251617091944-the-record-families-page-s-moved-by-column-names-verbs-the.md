---
schema_version: 1
id: "iss-2609251617091944"
slug: "the-record-families-page-s-moved-by-column-names-verbs-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md"
resolution: "The Moved by column names only verbs the binary carries, and marks the step's and the bundle's verbs as the planned intents that bring them; the bundle entry no longer gives the --bundle form as present."
impact: internal
resolved_by:
  commit: "9f75c19d35242d83750e50a64b85779322e14281"
---

The record-families page's Moved by column names verbs the binary does not have: intent reclassify, abcd build, intent plan --bundle and drain are not in go run ./cmd/abcd --help, and the bundle glossary entry claims abcd intent plan itd-A itd-B --bundle <name> as present; itd-34 (multi-intent plan) and itd-2609212103565953 (steps and the build) are both planned. The page every glossary entry points at teaches a reader commands that refuse.

## Grounds

- pursued: we expect a reader following the page to run only commands that exist; shown wrong if any verb the page names is refused as unknown by go run ./cmd/abcd
