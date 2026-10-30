---
schema_version: 1
id: "iss-2610262034403157"
slug: "itd-117-puts-the-machine-wide-rules"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 513 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-c08d53b7af281ffe
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same rename; the later shipped intent decides the machine layer lives under ~/.abcd.noindex/."
---

itd-117 puts the machine-wide rules layer at ~/.abcd/rules.json; the intent chapter puts the machine layer of layered configuration under ~/.abcd.noindex/.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:670` — "over the machine's `~/.abcd.noindex/oracle-routing.json`"

itd-117's press release has the user-scope conventions live once in ~/.abcd/rules.json, layered between the bundled defaults and the repository's .abcd/rules.json. 05-intent describes the same layering for the routing table: the invocation's override over the repository's file, over the machine's ~/.abcd.noindex/oracle-routing.json, over the bundled proposal. The machine scope is named ~/.abcd/ in one document and ~/.abcd.noindex/ in the other, so two readers of one layering pattern look in different homes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 513.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same rename; the later shipped intent decides the machine layer lives under ~/.abcd.noindex/.
