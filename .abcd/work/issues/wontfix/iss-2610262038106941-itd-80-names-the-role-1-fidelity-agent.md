---
schema_version: 1
id: "iss-2610262038106941"
slug: "itd-80-names-the-role-1-fidelity-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 776 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-d0bc639e1e41ed0e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same agent rename as the press-release record; the agents directory holds intent-auditor."
---

itd-80 names the Role 1 fidelity agent `intent-fidelity-reviewer`; 34-build gives the same closing-lane fidelity audit to `intent-auditor`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:35` — "A host-delegated `intent-fidelity-reviewer` markdown agent"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:683` — "and, on the lane whose landing closes the spec, an `intent-auditor`"

itd-80 specifies a host-delegated `intent-fidelity-reviewer` agent that reads an intent's acceptance criteria and the delivered diff and returns the per-criterion verdict the spec close ingests. 34-build hands that same audit, whose verdict the landing's close consumes, to an agent named `intent-auditor`. One agent carries two names across the record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 776.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same agent rename as the press-release record; the agents directory holds intent-auditor.
