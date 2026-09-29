---
schema_version: 1
id: "iss-110"
slug: "agents-changelog-md-and-agents-readme-md-are-plain-docs-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "agents/README.md and agents/CHANGELOG.md move to .abcd/development/agents/, outside the loader's root, so the installed plugin no longer lists abcd:README and abcd:CHANGELOG as agents; agent_contract reads the log from its configured changelog path, and every live reference follows. TestPluginAgentSurfaceRegistersOnlyAgents refuses any markdown file at the top of agents/ that is not a prompt naming itself."
impact: fix
resolved_by:
  commit: "d5091dac6"
---

agents/CHANGELOG.md and agents/README.md are plain docs (the itd-5 prompt-version log; a readme) but the plugin agent-loader globs agents/*.md, so both are mis-registered as agents (abcd:CHANGELOG, abcd:README appear in the harness agent list). They have no agent frontmatter and are not invokable workers. Fix: either move these docs out of agents/ (e.g. to .abcd/development/ or a docs path) or make the loader skip non-agent files (require agent frontmatter). Surfaced by the derived-changelog plan adversarial review, which had assumed the abcd:CHANGELOG slot was free for a new composer agent.

## Grounds

- pursued: every markdown file at the top of agents/ is a prompt, so the harness registers only real agents; an installed surface still listing abcd:README or abcd:CHANGELOG after this release would show it wrong
