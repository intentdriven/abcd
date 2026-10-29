---
schema_version: 1
id: "iss-2610262035555270"
slug: "the-out-of-scope-list-holds-abcd-can"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 609 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:68"
remedy: "Supersede itd-86 by the shipped cold-reading intents (itd-184 and itd-185) after checking nothing in its scope remains, so the generated out-of-scope list drops it."
finding_ref: cfk-73730d0693fcfecd
---

The out-of-scope list holds 'abcd Can Read Its Own Design Documents As A Stranger Would' as later-phase work; the reading chapter ships exactly that outside reading of the design

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:68` — "abcd Can Read Its Own Design Documents As A Stranger Would"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:3` — "A cold reading is an outside look at the design by a reader who has not been told what the project already believes"

03-out-of-scope.md lists itd-86, 'abcd Can Read Its Own Design Documents As A Stranger Would', among the uncommitted drafts. 23-reading.md describes the shipped cold-reading instrument as an outside look at the design by a reader who has not been told what the project already believes. The same ground is claimed as shipped by one document and as unscheduled later-phase work by the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 609.
