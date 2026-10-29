---
schema_version: 1
id: "iss-2610262035378747"
slug: "the-land-chapter-calls-itd-97-s-auto"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 585 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-land.md:121"
remedy: "none (filed automatically)"
finding_ref: cfk-8e1299b5e126435d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262036211949. 38-land's 'facilitator mode' (itd-97's merge toggle, of which itd-2610150602547271 is the first slice) collides with the same facilitator value of abcd mode."
duplicates: [iss-2610262036211949]
---

The land chapter calls itd-97's auto/human merge toggle 'the facilitator mode'; the mode chapter uses `facilitator` as a value of `abcd mode`'s waiting-on state.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/38-land.md:121` — "seat of the facilitator mode draft itd-97 describes (auto: queue at once;"
- End B: `.abcd/development/brief/04-surfaces/26-mode.md:4` — "waiting-on state behind the status line's badge: `managed`, `facilitator` or"

38-land names the merge gate the seat of 'the facilitator mode' itd-97 describes, an auto/human toggle under which every repository runs in 'human mode' for now. 26-mode's `abcd mode` holds a waiting-on state whose values are `managed`, `facilitator` and `product-thinker`, plus a separate away mode. So 'facilitator mode' names two concepts: a merge toggle, and the mode state that says the loop waits on the facilitator. Its partner 'human mode' reads like the badge parked on a person.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 585.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262036211949. 38-land's 'facilitator mode' (itd-97's merge toggle, of which itd-2610150602547271 is the first slice) collides with the same facilitator value of abcd mode.
