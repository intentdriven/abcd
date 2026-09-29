---
schema_version: 1
id: "iss-241"
slug: "research-prompting-agents-holds-prompt-baselines-for-chat-di"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: ".abcd/development/research/prompting/agents"
resolution: "research/prompting/agents/README.md's inventory states each research file's standing against agents/: chat-distiller and embark-scaffolder are design targets the brief (05-internals/01-agents.md) lists as still to be built, never shipped prompts, and intent-fidelity-reviewer.md covers the shipped intent-auditor under its earlier name, which a dated section on that file records; prompting/README.md says the same. No gate would have caught it: nothing compares the research inventory with agents/."
impact: internal
resolved_by:
  commit: "29d4ef090"
---

research/prompting/agents/ holds prompt baselines for chat-distiller and embark-scaffolder, neither of which exists under agents/ any more — the baseline corpus has drifted from the shipped agent set.

## Grounds

- pursued: every research file under research/prompting/agents/ names an agent that ships under agents/ or one the brief's roster lists as still to be built, and the inventory says which; a research file whose agent is neither, or an inventory row claiming a shipped prompt that agents/ lacks, would show it wrong
