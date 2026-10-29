---
schema_version: 1
id: "iss-2610262038460690"
slug: "itd-11-has-lifeboat-oracle-audit-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 817 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-11]
finding_ref: cfk-83b7b389ee72b809
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032561511. Identical A end and the same lifeboat-reviewer fact, cited from the agents chapter."
duplicates: [iss-2610262032561511]
---

itd-11 has lifeboat-oracle audit a lifeboat's output; 01-agents names the lifeboat auditor lifeboat-reviewer

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15` — "the lifeboat-oracle's audit catches and quarantines low-confidence Pass B output before it pollutes the brief"
- End B: `.abcd/development/brief/05-internals/01-agents.md:23` — "`lifeboat-reviewer` (`disembark review`), `graveyard-interpreter`"

itd-11 has lifeboat-oracle audit Pass B output and quarantine what fails before it reaches the lifeboat brief. 01-agents names the agent that audits a packed lifeboat lifeboat-reviewer, serving disembark review, and lists no lifeboat-oracle. The same role has two names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 817.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032561511. Identical A end and the same lifeboat-reviewer fact, cited from the agents chapter.
