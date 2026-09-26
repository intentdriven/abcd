---
schema_version: 1
id: "iss-2609261039132350"
slug: "a-principle-whose-statement-is-written-as-an-h2-heading-the"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-principles"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "A principle's statement resolves by its **The rule.** paragraph alone, through lint.FindPrincipleStatement shared with principle_claims; a ## The rule section never travels, and a typed heading-shaped principle is reported by principle_claims."
impact: fix
resolved_by:
  commit: "a5f5a266662ed258f549a49a4847a391e68fd9f6"
---

A principle whose statement is written as an H2 heading (## The rule) is projected by projectField's heading leg before the labelled-paragraph leg, so the whole section travels to a reading, its Why and Bounds included, under a manifest naming the field The rule; the lint judges only the labelled paragraph, so no gate refuses the shape and the floor's statement-alone promise breaks on a file shape.

## Grounds

- pursued: the LEAKHEAD probe (TestHeadingShapedStatementNeverTravels) carries no Why or Bounds at any cold position and TestHeadingShapedPrincipleHasNoStatement reports the typed shape; a bundle holding LEAKHEAD-WHY, or a typed heading-shaped principle passing record-lint, would show it wrong
