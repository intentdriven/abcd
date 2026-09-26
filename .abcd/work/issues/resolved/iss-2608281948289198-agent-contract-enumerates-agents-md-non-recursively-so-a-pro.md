---
schema_version: 1
id: "iss-2608281948289198"
slug: "agent-contract-enumerates-agents-md-non-recursively-so-a-pro"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-151 security review"
found_at: "internal/core/lint/agentcontract.go"
resolution: "agent_contract walks the agents tree and refuses any markdown file below the top level outside a fixtures/ directory as a misfiled prompt; agents/README.md states the flat layout and the refusal. TestAgentContractRefusesANestedPrompt pins one-level and deeper nesting and leaves a fixture alone."
impact: internal
resolved_by:
  commit: "cd7c4356"
---

agent_contract enumerates agents/*.md non-recursively, so a prompt filed at agents/<name>/<name>.md is skipped entirely — it draws no trust-contract finding, no canary demand and no changelog demand, which is a silent opt-out of the whole rule by choosing a directory. The flat layout is the documented convention (agents/README.md), so this is a gap rather than a supported layout: either refuse a markdown file found one level down as a misfiled prompt, or walk the tree. Found by adversarial security review of itd-151.

## Grounds

- pursued: no prompt escapes the trust contract by its directory; a markdown file under agents/<name>/ outside fixtures/ that record-lint does not name would show it wrong
