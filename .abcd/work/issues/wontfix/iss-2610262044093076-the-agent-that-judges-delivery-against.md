---
schema_version: 1
id: "iss-2610262044093076"
slug: "the-agent-that-judges-delivery-against"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1187 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-30, itd-24]
finding_ref: cfk-97af38718a14e961
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034207649. Same draft edit as iss-2610262034207649, set against a different statement of the same current state."
duplicates: [iss-2610262034207649]
---

The agent that judges delivery against an intent is `intent-fidelity-reviewer` in itd-30 and the intent auditor in itd-24

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`intent-fidelity-reviewer` reads either dialect against the same delivered reality"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:61` — "judging delivered reality against each intent's criteria is the intent auditor's job"

itd-30 has `intent-fidelity-reviewer` read both press releases and design fictions against delivered reality. itd-24 says judging delivered reality against an intent's criteria is the intent auditor's job. The two documents use different names for the same agent, and a builder of itd-30 would look for an agent that does not exist under that name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1187.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034207649. Same draft edit as iss-2610262034207649, set against a different statement of the same current state.
