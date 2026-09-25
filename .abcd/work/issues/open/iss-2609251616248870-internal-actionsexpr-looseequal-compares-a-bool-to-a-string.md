---
schema_version: 1
id: "iss-2609251616248870"
slug: "internal-actionsexpr-looseequal-compares-a-bool-to-a-string"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

internal/actionsexpr looseEqual compares a bool to a string by truthiness, where GitHub Actions coerces both to numbers (true == 'true' is false on GitHub), so an if: expression making that comparison would evaluate differently in the test evaluator than in the runner (actionsexpr.go:424-429; carried over unchanged from the old lint evaluator; no committed workflow makes that comparison today; review2-workflows R1).
