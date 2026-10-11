---
schema_version: 1
id: "iss-2608220149008905"
slug: "pull-request-target-checkout-guard"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: ".github/workflows/external-review.yml"
resolution: "workflow-walking test asserts no pull_request_target workflow references actions/checkout"
impact: internal
resolved_by:
  commit: "ae07282"
---

external-review.yml suppresses zizmor dangerous-triggers on its pull_request_target trigger, so the safety invariant it states in prose (PR code never checked out) is unenforced; a later checkout added to that job passes every zizmor persona, a pwn request; add a Go test asserting no pull_request_target workflow checks out code