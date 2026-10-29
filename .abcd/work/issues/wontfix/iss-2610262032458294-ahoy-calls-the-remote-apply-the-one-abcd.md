---
schema_version: 1
id: "iss-2610262032458294"
slug: "ahoy-calls-the-remote-apply-the-one-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 313 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:90"
remedy: "none (filed automatically)"
finding_ref: cfk-f5d8c97424387890
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032226321. Same false 'one verb mutates state outside this machine' claim; this record cites the site setup's forge and hosting writes."
duplicates: [iss-2610262032226321]
---

ahoy calls the remote apply the one abcd verb that mutates state outside the machine; 10-site has abcd site setup configuring forge environments through gh and the host through the hosting adapter

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:90` — "the one abcd verb that mutates state outside this machine"
- End B: `.abcd/development/brief/05-internals/10-site.md:88` — "The forge environments go through `gh` as the invoking person, and the host through the hosting adapter seam"

01-ahoy asserts that the remote apply is the one abcd verb that mutates state outside this machine. 10-site describes abcd site setup sending the forge environments through gh as the invoking person and the provider's host configuration through the hosting adapter seam, both writes to remote services. Either site setup writes nothing remote or the remote apply is not the only outward-mutating verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 313.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032226321. Same false 'one verb mutates state outside this machine' claim; this record cites the site setup's forge and hosting writes.
