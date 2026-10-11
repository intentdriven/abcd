---
schema_version: 1
id: "iss-2610072113549994"
slug: "a-user-visible-change-reached-main-with"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "release preparation, 2026-10-07: a user-visible change merged with no record"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question.go"
remedy: "Resolve against the merge that shipped it (12a5dca2f) with impact additive, so the derived changelog announces it."
resolution: "Shipped by 12a5dca2f (the guard's JSON deny, the rows-only note, abcd:question-drafter); recorded at release preparation so the changelog carries it."
impact: additive
resolved_by:
  commit: "12a5dca2f"
---

A user-visible change reached main with no record, so the derived changelog would not announce it: the guard hook refuses through the host's JSON deny, so a refused command no longer echoes the hook's command line before the guard's message; a question whose only fault is its length is shown with a note to the agent rather than refused; and a new abcd:question-drafter agent drafts abcd's questions so they fit. Merged as 12a5dca2f; iss-2610070637562567 stays open because the host still draws every hook refusal as an error.
