---
schema_version: 1
id: "iss-2609261140284421"
slug: "setext-principle-title-not-carried"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-principles LOW"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/principles.go"
resolution: "FindPrincipleStatement reads a setext H1 (the paragraph above a === underline) as well as an ATX one, so the projection carries the title and principle_claims judges it"
impact: fix
resolved_by:
  commit: "0e9a7c5b9"
---

A principle whose H1 title is written in setext form (the title line underlined with ===) travels to a reading without its title: principleTitleRe in internal/core/lint/principles.go:127 matches ATX headings only, so the projection sends the statement paragraph bare, and a cold reader gets a sentence with no name. Both readers agree, so no gate disagrees; the promise that a principle is readable cold does not hold for that file shape.

## Grounds

- pursued: a principle titled in setext form now travels to a reading with its title above the statement; a setext-titled principle projected bare, or a citation in a setext title passing principle_claims, would show it wrong
