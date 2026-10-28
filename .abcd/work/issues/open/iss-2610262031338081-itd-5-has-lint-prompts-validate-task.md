---
schema_version: 1
id: "iss-2610262031338081"
slug: "itd-5-has-lint-prompts-validate-task"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 145 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-5]
finding_ref: cfk-9a3c6bc2567d7f62
---

itd-5 has lint_prompts validate task_classes against an enum in internal/core/lint; the naming register says the binary carries no such schema

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:23` — "`lint_prompts` validates set-membership against the `task_classes` enum in `internal/core/lint`"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:189` — "This table is the source of truth for the values: the binary carries no `task_classes` schema"

itd-5's rule says lint_prompts validates set-membership against the task_classes enum in internal/core/lint. The reserved-vocabulary register says its own table is the source of truth because the binary carries no task_classes schema. The discipline claims an enforcement the register says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 145.
