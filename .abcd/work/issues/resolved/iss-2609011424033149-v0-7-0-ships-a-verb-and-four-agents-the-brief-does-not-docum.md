---
schema_version: 1
id: "iss-2609011424033149"
slug: "v0-7-0-ships-a-verb-and-four-agents-the-brief-does-not-docum"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "the v0.7.0 cut, iss35 full-tier crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md"
resolution: "the reading verb and the four cold-reading definitions are documented: 23-reading carries the size report and the delivering records, 01-agents carries the four definitions with their position and regime fields and the corrected counts, and 08-skills, 08-abcd, 03-configuration and the surfaces index count them"
impact: fix
resolved_by:
  commit: "b18c7c5d8eaf7e1153d106f5f4432b0595b25641"
---

v0.7.0 ships a verb and four agents the brief does not document anywhere

## Grounds

- pursued: every claim added to the brief is checkable against the built binary, a command page or an agent definition, so the next crosscheck receipt finds no release-introduced finding against v0.7.0's surface; a re-run of the iss35 crosscheck at this commit reporting any of the fifteen findings would show it wrong
