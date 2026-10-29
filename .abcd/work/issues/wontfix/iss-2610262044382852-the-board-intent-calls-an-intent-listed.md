---
schema_version: 1
id: "iss-2610262044382852"
slug: "the-board-intent-calls-an-intent-listed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1220 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610031214560142-when-you-type-abcd-in-a-terminal-or-ask.md:97"
remedy: "none (filed automatically)"
related_intents: [itd-2610031214560142, itd-2609212103568351]
finding_ref: cfk-ef71f031e16f9054
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The tree no longer lists the head twice, so the planned board intent's decision 9 and the current brief agree; only the shipped intent's original scope sentence is out of date history."
---

The board intent calls an intent listed under both Now and Next a defect, but the shipped status block lists the pick head in both by construction

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610031214560142-when-you-type-abcd-in-a-terminal-or-ask.md:97` — "the next-up intent listed under both Now and Next is the defect's (iss-2610031207397996)"
- End B: `.abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:39` — "plus the head of `build next`'s pick order marked "next up", so Now is never empty while anything is READY"

itd-2610031214560142 treats the next-up intent appearing under both Now and Next as a defect (iss-2610031207397996). itd-2609212103568351 defines Now to include the head of build next's pick order marked 'next up', and Next as every READY intent in pick order, so the head is necessarily in both. The shipped design produces what the planned record calls a defect.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1220.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The tree no longer lists the head twice, so the planned board intent's decision 9 and the current brief agree; only the shipped intent's original scope sentence is out of date history.
