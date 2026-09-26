---
schema_version: 1
id: "iss-2609261117268064"
slug: "testeveryrangescopedgateresolvesabaseoneveryevent-s"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-daport"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/rangegatebase_test.go"
resolution: "The delegation exemption is an exact run line: a step earns it only when its single single-line run: equals an allowed gate invocation, so a second, unguarded BASE_SHA use beside the allowed command no longer passes."
impact: internal
resolved_by:
  commit: "3689eb48"
---

TestEveryRangeScopedGateResolvesABaseOnEveryEvent's delegation exemption (internal/core/lint/rangegatebase_test.go, baseGuardingGates and delegatesBaseGuard) is a substring test over the whole step body: a ci.yml step that keeps the literal go run ./cmd/record-lint decisions-append "$BASE_SHA" and also hands BASE_SHA to a second, unguarded command (appended with &&, on a line of its own, or with the literal only in a comment or an echo) still earns the exemption, so the test stops asking that step to refuse the all-zeroes placeholder. Fix: the step earns it only when its whole run script is exactly an allowed gate invocation.

## Grounds

- pursued: TestBaseGuardDelegationCoversTheWholeStep plants a second use on the same line, on its own line, with braces, inside an echo and behind a comment, and requires each to lose the exemption; five of its six cases failed before the fix. A step with an unguarded second BASE_SHA use that still passes the sweep would show it wrong.
