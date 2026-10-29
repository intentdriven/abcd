---
schema_version: 1
id: "iss-2610262032561511"
slug: "itd-11-gives-the-lifeboat-audit-to-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 338 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15"
remedy: "In itd-11 (press release, scope item and acceptance criterion 3) replace 'lifeboat-oracle's audit' with lifeboat-reviewer's review (disembark review), the agent and the word itd-125 settled"
related_intents: [itd-11]
finding_ref: cfk-de28ee82b65f569c
---

itd-11 gives the lifeboat audit to a lifeboat-oracle; disembark's lifeboat review agent is lifeboat-reviewer, and no lifeboat-oracle appears

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15` — "the lifeboat-oracle's audit catches and quarantines low-confidence Pass B output"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:178` — "the principles to `principle-distiller` and the review to `lifeboat-reviewer`"

itd-11 relies on 'the lifeboat-oracle's audit' to quarantine low-confidence Pass B output. The disembark chapter names the four agents its synthesis steps hand work to, and the one that reviews a lifeboat is `lifeboat-reviewer`. No agent by the other name appears. Either one agent goes by two names, or the draft depends on an agent the surface does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 338.
