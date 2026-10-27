---
schema_version: 1
id: "iss-2610251559204442"
slug: "disembark-probe-plan-and-pack-share-one"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Carry the wide-scan scope into disembark plan's manifest and its text render, as the coverage report does, so the shared help holds for all three verbs."
---

disembark probe, plan and pack share one --include-ignored help (internal/surface/cli/cli.go:1332; docs/reference/cli/commands.md:1002, 1020, 1070), and it promises '(widens the scan; the report says so)'. probe's coverage report prints 'scope: WIDE' (internal/core/lifeboat/coverage.go:114-116), and pack writes that report into the lifeboat (plan.go:226-230). plan's output, text and --json alike, is the PlanManifest (plan.go:402-410; RenderManifest 434-450), which carries no scope, so abcd disembark plan --include-ignored says nothing of the wider scan.
