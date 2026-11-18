---
schema_version: 1
id: "iss-2611181522116390"
slug: "testareadgrowthtestrefusesaquadraticread"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "merge-queue pre-check of a records-only pull request"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/findings_test.go"
remedy: "Follow the remedy readGrowth's comment names: make the control's quadratic read four times longer again (8,000 a side), leaving the median judge, its five rounds and quadraticGrowth unchanged, and measure the test's time plain and under -race so the added cost is known; if a third flake follows, change the judge, not the size"
---

TestAReadGrowthTestRefusesAQuadraticRead in internal/core/implement/loop failed again, in the plain test step of the ubuntu check job of a records-only pull request on 2026-11-18, after iss-2611152052517498 made its quadratic read four times longer: the control grew 9.8 times on an input four times larger against a threshold of more than 10, with the whole module's tests running beside it. The fix recorded there held under -race on a quiet machine but not on a loaded runner, and the readGrowth comment names the same remedy for a second flake.
