---
schema_version: 1
id: "iss-278"
slug: "five-agent-prompts-read-attacker-influenceable-input-without"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "manual-capture"
found_at: "agents/"
related_intents: [itd-151]
resolution: "record-lint's agent_contract rule enforces the itd-5 trust contract over agents/: the trust-contract frontmatter, the injection-canary fixture, and a per-agent changelog entry over a diff (itd-151)"
impact: internal
resolved_by:
  intent: "itd-151"
  spec: "spc-44"
---

Five agent prompts read attacker-influenceable input without the itd-5 contract (ruthless-reviewer, security-reviewer, docs-currency-reviewer, intent-auditor, sota-researcher) and agents/ sits outside both lint roots, so no detector exists for the class; the PQ linter (agents/README.md) is the missing detector