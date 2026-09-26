---
schema_version: 1
id: "iss-2609251734057081"
slug: "exclusive-flag-group-exits-one"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "markUsageErrorsExitTwo gives every command a PreRunE that runs cobra's flag-group check first and tags its refusal exit 2; TestEveryExclusiveFlagGroupRefusesAsAUsageError holds every declared group to it in both renders."
impact: fix
resolved_by:
  commit: "f43a4217"
---

A mutually exclusive flag group refuses with exit 1, not the usage-error exit 2: cobra's group validation runs after PreRunE and returns a plain error that markUsageErrorsExitTwo never tags, so abcd ahoy --dry-run --identity, abcd update --check --yes and abcd site build --preview --version exit 1, which a gate reads as a finding rather than a mis-spelt invocation

## Grounds

- pursued: every mutually exclusive group refuses with exit 2 wherever it is declared; shown wrong by any group whose two-flag invocation exits other than 2, which the tree walk would name
