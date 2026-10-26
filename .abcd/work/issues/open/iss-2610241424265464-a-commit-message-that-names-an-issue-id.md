---
schema_version: 1
id: "iss-2610241424265464"
slug: "a-commit-message-that-names-an-issue-id"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "records branch for itd-2610240922461281, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/commit-msg"
remedy: "Run the message-only half of RS004 (every issue id the message names has a Refs: or Resolves: declaration) in the commit-msg hook, beside abcd lint outbound, so the refusal comes at the commit; keep RS004 in preflight and CI for the range."
---

A commit message that names an issue id without a Refs: or Resolves: line passes the commit-msg hook and is refused only by lint-issues RS004 inside make preflight. On 2026-10-24 that cost a whole preflight run (tens of minutes under the load gate) for a one-line amend, on a machine where preflight capacity was the scarce resource.
