---
schema_version: 1
id: "iss-2610262038443634"
slug: "itd-5-counts-14-now-15-agents-under-its"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 815 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25"
remedy: "Replace 'abcd ships all 14 (now 15) agents' in itd-5 line 25 with 'every agent prompt in agents/', with no count"
related_intents: [itd-5]
finding_ref: cfk-af6ef4926a7aab85
resolution: "itd-5 line 25 now says abcd ships every agent prompt in agents/ under its rule, with no count; a reachaudit test refuses a shipped-agent count on the page."
impact: internal
resolved_by:
  commit: "6524003629d7dfbca62b2af38e9448da1b2e3f1a"
---

itd-5 counts 14 (now 15) agents under its rule; 01-agents says eighteen agent prompts ship

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25` — "abcd ships all 14 (now 15) agents under this rule."
- End B: `.abcd/development/brief/05-internals/01-agents.md:18` — "Eighteen agent prompts ship in `agents/` today, in five groups:"

itd-5 says abcd ships 14 (now 15) agents under the discipline. 01-agents says eighteen agent prompts ship in agents/ today. The discipline's count of the roster it governs is wrong.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 815.

## Grounds

- pursued: itd-5 carries no count of the shipped agents, so it cannot contradict 01-agents' roster as it grows; TestPromptQualityDisciplineDoesNotCountTheAgents would show it wrong by finding a 'ships [all] N agents' claim or missing the 'every agent prompt in agents/' wording.
