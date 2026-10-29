---
schema_version: 1
id: "iss-2610262040083517"
slug: "the-agent-catalog-says-the-auditor-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 911 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:137"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-bab72f253be966e5
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent says the shape role is not delivered, which 01-agents.md states."
---

The agent catalog says the auditor's third role, kind classification, is a design target; shipped itd-48 announces all three roles live, /abcd:intent shape included

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:137` — "The first two ship; the third is a design target."
- End B: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "With all three roles live, `/abcd:intent consistency` and `/abcd:intent shape` move from documented command surface to working command surface."

01-agents says that of the auditor's three roles the first two ship and the third, kind classification through `abcd intent shape`, is a design target with no shape sub-verb registered. itd-48, in shipped/, says it ships Role 3 and that with all three roles live /abcd:intent shape becomes a working command surface. The shape role cannot be both live and unbuilt.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 911.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent says the shape role is not delivered, which 01-agents.md states.
