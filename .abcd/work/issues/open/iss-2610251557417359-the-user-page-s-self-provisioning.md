---
schema_version: 1
id: "iss-2610251557417359"
slug: "the-user-page-s-self-provisioning"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hooks_selfprovision_count_test.go"
remedy: "Hold docs/how-to/install.md's self-provisioning paragraph to the manifest's salvaging set as the brief chapters are, so a hook that gains or loses its bootstrap rung fails until the page says so."
---

The user page's self-provisioning paragraph is held to nothing. TestSelfProvisionParagraphsMatchTheShippedSets and TestTheBriefNamesSessionEndAsTheBootstrapException read only the two brief chapters (internal/surface/cli/hooks_selfprovision_count_test.go:74-85, hooks_selfprovision_test.go:410-443), so 5d754f95b updated the brief when it wired Stop and StopFailure without a bootstrap rung, and docs/how-to/install.md:79-91 kept naming session end as the only hook that never downloads.
