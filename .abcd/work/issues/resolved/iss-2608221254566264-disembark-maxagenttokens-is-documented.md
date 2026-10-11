---
schema_version: 1
id: "iss-2608221254566264"
slug: "disembark-maxagenttokens-is-documented"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "context-window SOTA investigation"
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
resolution: "Already true at base: since 53a5a913e (v0.10.0) the configuration chapter lists disembark.maxAgentTokens under Staged config keys, which no shipped code reads, so the brief no longer describes it as live. The one remaining citation, in the meta chapter, now names it as the staged key it is."
impact: internal
resolved_by:
  commit: "459a317d9"
---

disembark.maxAgentTokens is documented in the brief (05-internals/03-configuration.md) as a per-agent context budget with stream+summarise overflow behaviour, but no code reads the key and it is absent from .abcd/config.json — brief-vs-binary drift.

## Grounds

- pursued: the brief names maxAgentTokens only as a staged, unread key; a brief passage describing it as a budget in force would show it wrong
