---
schema_version: 1
id: "iss-2610262038425929"
slug: "itd-5-calls-the-prompt-linter-lint"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 813 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:23"
remedy: "In itd-5's rule text, Add 1 and Add 4, rename lint_prompts to agent_contract (the shipped record-lint rule) and state that set-membership of task_classes against a closed enum, the PQ005 block-list code and the task_classes.json enum are staged (iss-265); the shipped check only requires the field to be non-empty"
related_intents: [itd-5]
finding_ref: cfk-ab4940e4499b0676
---

itd-5 calls the prompt linter lint_prompts; 05-prompt-quality calls the one shipped prompt-tree rule agent_contract

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:23` — "`lint_prompts` validates set-membership against the `task_classes` enum in `internal/core/lint`"
- End B: `.abcd/development/brief/05-internals/05-prompt-quality.md:26` — "`agent_contract`, in `internal/core/lint`, is record-lint's dedicated rule for the agent-prompt tree."

itd-5 says the lint_prompts check validates task_classes set-membership in internal/core/lint. 05-prompt-quality says agent_contract is record-lint's dedicated rule for the agent-prompt tree, and that the membership check is staged. One linter has two names, and the two documents also disagree about what it checks.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 813.
