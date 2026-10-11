---
schema_version: 1
id: "iss-2609251616248870"
slug: "internal-actionsexpr-looseequal-compares"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "actionsexpr's == follows GitHub's documented loose-equality table: mismatched types coerce to numbers (null 0, booleans 1/0, strings by the JSON number grammar with empty as 0 else NaN, arrays and objects NaN), NaN equals nothing, strings compare ignoring case; TestLooseEqualFollowsGitHubsCoercionTable pins the table."
impact: internal
resolved_by:
  commit: "e9d2c9b59"
---

internal/actionsexpr looseEqual compares a bool to a string by truthiness, where GitHub Actions coerces both to numbers (true == 'true' is false on GitHub), so an if: expression making that comparison would evaluate differently in the test evaluator than in the runner (actionsexpr.go:424-429; carried over unchanged from the old lint evaluator; no committed workflow makes that comparison today; review2-workflows R1).

## Grounds

- pursued: the test evaluator answers every == the runner would answer the same way; a row of the documented table evaluating differently would show it wrong
