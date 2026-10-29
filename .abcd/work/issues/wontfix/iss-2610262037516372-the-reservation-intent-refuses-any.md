---
schema_version: 1
id: "iss-2610262037516372"
slug: "the-reservation-intent-refuses-any"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 754 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:162"
remedy: "none (filed automatically)"
related_intents: [itd-2609091034175565]
finding_ref: cfk-32e6d6b96282d20a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The intent's In Scope line for build is unqualified and its decision 31 and criterion ac-21 are about the reserve act; the code makes the two behaviours different on purpose."
---

The reservation intent refuses any remote layout without origin; the build chapter reserves nothing in such a checkout and none of its checks refuses the run.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:162` — "In a checkout with no remote named `origin` nothing is reserved."
- End B: `.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md:253` — "Any other remote layout is refused, naming the remotes found."

itd-2609091034175565 decision 31, and its In Scope 'Only origin' line, refuse any other remote layout, naming the remotes found, and its spec 2 has abcd build reserve the intent when it starts. 34-build.md says that in a checkout with no remote named origin nothing is reserved, and no build check refuses the run for it. 27-implement.md and 35-drain.md say the same of the land fence and the drain. A build in a checkout without origin is refused by one document and runs unreserved by the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 754.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The intent's In Scope line for build is unqualified and its decision 31 and criterion ac-21 are about the reserve act; the code makes the two behaviours different on purpose.
