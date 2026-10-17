---
schema_version: 1
id: "iss-2610160608434377"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md"
remedy: "Say at 01-agents.md:45 that the four repo-workflow reviewer and researcher prompts return prose or a table, and the rest declare JSON."
resolution: "the agents chapter now says fourteen prompts return JSON and the four repo-workflow reviewer and researcher prompts return a table, prose with a verdict, or a ranked list"
impact: internal
---

The brief at .abcd/development/brief/05-internals/01-agents.md:45 says each of the eighteen agent prompts declares its inputs and outputs as JSON. agents/docs-currency-reviewer.md asks for a report as a table, agents/ruthless-reviewer.md and its companion code reviewer return prose verdicts, and agents/sota-researcher.md holds no JSON at all, so four prompts declare no JSON contract.
