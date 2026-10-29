---
schema_version: 1
id: "iss-2610262033205781"
slug: "itd-48-calls-the-lifeboat-review-agent"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 391 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:120"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-b0788476e932d546
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. agents/ holds lifeboat-reviewer only and spc-30 records the rename, so there is one lifeboat review agent."
---

itd-48 calls the lifeboat review agent `lifeboat-oracle`; 02-disembark calls it `lifeboat-reviewer`

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:120` — "**`lifeboat-oracle` agent.** itd-5's named reviewer; out of scope here."
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:179` — "and the review to `lifeboat-reviewer`"

itd-48 names itd-5's lifeboat reviewer the `lifeboat-oracle` agent. 02-disembark sends the lifeboat review to `lifeboat-reviewer`, the agent that returns the registered verdict. With two names for one agent, a reader cannot tell whether a second lifeboat review agent exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 391.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. agents/ holds lifeboat-reviewer only and spc-30 records the rename, so there is one lifeboat review agent.
