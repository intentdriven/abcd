---
schema_version: 1
id: "iss-2608281752471145"
slug: "secret-shaped-test-fixtures-committed"
severity: "major"
category: "process"
source: "impl-review"
found_during: "gitleaks CI failure triage (2026-08-28)"
found_at: "internal/adapter/gitleaks/gitleaks_test.go"
resolution: "resolved in the clean-history follow-ups branch"
impact: internal
resolved_by:
  commit: "97c6d6d7"
---

the opt-in gitleaks adapter's test fixtures commit a verbatim secret-shaped literal (a 40-char high-entropy generic-api-key value) in internal/adapter/gitleaks/gitleaks_test.go and internal/core/history/gitleaks_augment_test.go, which trips the full-history gitleaks CI scan (gitleaks git, fetch-depth 0) and fails the PR and every PR batched with it in the merge queue. Secret-shaped test fixtures must be CONSTRUCTED AT RUNTIME, never committed verbatim, so no literal ever enters git history for the full-history scan to find. Fix: a shared runtime generator produces the synthetic high-entropy value; the two fixtures call it; no literal in source. Because the literal is already committed on the branch and history rewrite is off the table (never force-push), the fix lands on a fresh branch where the literal never entered history. Also record the doctrine as a durable principle.