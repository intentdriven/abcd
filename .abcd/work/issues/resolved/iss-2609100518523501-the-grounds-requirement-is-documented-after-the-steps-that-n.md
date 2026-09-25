---
schema_version: 1
id: "iss-2609100518523501"
slug: "the-grounds-requirement-is-documented-after-the-steps-that-n"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
resolution: "the grounds refusal it documents is parked (DECISIONS 2026-09-09, 667a57f4): intent ready reports grounds as advisory, so no reader meets a refusal first"
impact: internal
resolved_by:
  commit: "667a57f4fad4198182c01e82d5b7106474064dd6"
---

The grounds requirement is documented after the steps that need it, so a reader following the surface page in order meets the refusal before the explanation. The page presents creation and planning as a sequence and states the grounds requirement further down, among the interview steps. An agent working through it in order therefore runs the plan verb, is refused for missing grounds, and only then reaches the passage that would have told it why and how. The cost is small per encounter and it recurs for every reader who trusts the order of the page, which is every reader following it for the first time. The fix is ordering rather than content: state the requirement where the step that enforces it is described.

## Grounds

- pursued: a reader following commands/intent.md in order meets no grounds refusal; shown wrong if intent plan or ready refuses a missing grounds entry again
