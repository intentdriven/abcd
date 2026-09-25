---
schema_version: 1
id: "iss-2609251827290265"
slug: "the-launch-dry-run-s-plain-render-stages-only-two-gate-rows"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "The plain dry-run prints every gate row whose status is not ran, with its name, status and detail (TestLaunchDryRunPlainRenderStagesEveryUnranRow)."
impact: fix
resolved_by:
  commit: "59f7443e"
---

The launch dry-run's plain render stages only two gate rows loudly: internal/surface/cli/cli.go prints the citation-baseline and semantic-receipts rows and no other, so a documentation-auditor row that is not_armed (no .abcd/docs-lint.json) is visible only in --json and the pre-flight report file. iss-2608231226342272 is the precedent that a row only --json shows is invisible. Remedy: the plain render prints every gate row whose status is not 'ran'.

## Grounds

- pursued: every row --json reports as not ran appears in the plain render by name and status; a not_armed documentation-auditor row missing from the plain output would show it wrong
