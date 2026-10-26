---
schema_version: 1
id: "iss-2610252214199488"
slug: "05-intent-md-526-says-the-related-issues"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Name both checkers: issue drift and related_issues_backlink."
---

05-intent.md:526 says the related_issues/related_intents join is checked 'not by the record-lint'; related_issues_backlink is a record-lint rule armed at warn (internal/core/lint/reciprocal.go:225, .abcd/record-lint.json:497), as :511 of the same page states.
