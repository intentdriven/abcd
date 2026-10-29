---
schema_version: 1
id: "iss-2610262044129717"
slug: "the-phone-dashboard-was-ruled-read-only"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1190 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-2610032150577708, itd-2610222029228127]
finding_ref: cfk-c2871678297dc8b4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The two records give the phone different surfaces; the later draft carries over a 2026-10-17 ruling that does not reverse the dashboard's read-only decision."
---

The phone dashboard was ruled read-only and declined answering questions; the phone checklist app exists to answer them

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:40` — "asked whether the dashboard can answer abcd's questions or only be read (read only; answer on Tailscale; decide later): read only."
- End B: `.abcd/development/intents/drafts/itd-2610222029228127-a-person-away-from-their-computer-can.md:19` — "A person away from their computer can follow every abcd session's checklist and answer the questions that would stop its work from their phone."

itd-2610032150577708 shows the product thinker what is waiting on them on their phone, and records a ruling that it cannot answer abcd's questions ('answer on Tailscale' was declined). itd-2610222029228127 covers the same ground, the person's phone showing what is waiting on them, but its purpose is answering the questions that would stop a session, through a separate web app. The two records give contradictory answers to whether abcd's questions can be answered from the phone, and neither refers to the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1190.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The two records give the phone different surfaces; the later draft carries over a 2026-10-17 ruling that does not reverse the dashboard's read-only decision.
