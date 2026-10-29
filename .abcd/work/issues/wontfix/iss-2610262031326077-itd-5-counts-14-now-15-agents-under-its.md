---
schema_version: 1
id: "iss-2610262031326077"
slug: "itd-5-counts-14-now-15-agents-under-its"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 143 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-5]
finding_ref: cfk-32ca7fa90f954082
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038443634. Same stale count in itd-5 line 25 against a document that says eighteen; agents/ holds eighteen prompts."
duplicates: [iss-2610262038443634]
---

itd-5 counts 14 (now 15) agents under its rule, while the press release counts eighteen

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25` — "abcd ships all 14 (now 15) agents under this rule."
- End B: `.abcd/development/brief/01-product/01-press-release.md:27` — "eighteen agents, a vendor-agnostic adapter seam"

The prompt-quality discipline says abcd ships all 14 (now 15) agents under this rule. The brief's press release lists eighteen agents among its plumbing. Both cannot describe the current roster, so a reader cannot tell how many agents the discipline binds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 143.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038443634. Same stale count in itd-5 line 25 against a document that says eighteen; agents/ holds eighteen prompts.
