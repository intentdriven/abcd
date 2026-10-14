---
schema_version: 1
id: "iss-2610141705129324"
slug: "the-brief-agents-chapter-says-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.1 release review, iss35 brief-surface crosscheck (Direction B), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md"
remedy: "Say which of these agents a verb dispatches (build validate, the launch release gate) and which only a person does."
---

The brief agents chapter says the repo-workflow reviewers and researchers (docs-currency-reviewer, ruthless-reviewer, security-reviewer, sota-researcher) are dispatched by a human rather than by a verb, but commands/build.md:368 has the implement loop validate step dispatch a ruthless-reviewer and a security-reviewer and record their verdicts, and commands/launch.md:900 runs docs-currency-reviewer as a release-gate pass whose receipts launch checks.
