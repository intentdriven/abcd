---
schema_version: 1
id: "iss-2610262043392051"
slug: "itd-1-calls-the-role-1-fidelity-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1153 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-48]
finding_ref: cfk-6efeaa379cc2e93b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same rename as the other records in this cluster; the shipped intent keeps the name it was written under and the changelog carries the rename."
---

itd-1 calls the Role 1 fidelity agent intent-auditor; itd-48 calls the same agent intent-fidelity-reviewer

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17` — "emitted by the `intent-auditor` agent's single-document role"
- End B: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "abcd's `intent-fidelity-reviewer` agent grows from one `role` to three."

itd-1 says the per-criterion verdicts in a shipped intent's Audit Notes come from the intent-auditor agent's single-document role. itd-48 gives the name intent-fidelity-reviewer to the agent whose Role 1 writes those same MET/NOT_MET verdicts. One agent has two names across the records, so a reader cannot tell from the corpus which agent file or dispatch name is the real one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1153.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same rename as the other records in this cluster; the shipped intent keeps the name it was written under and the changelog carries the rename.
