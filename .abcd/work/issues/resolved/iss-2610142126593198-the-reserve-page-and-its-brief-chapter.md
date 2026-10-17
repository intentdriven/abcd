---
schema_version: 1
id: "iss-2610142126593198"
slug: "the-reserve-page-and-its-brief-chapter"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "the review of steps 8 to 11 and 14 of spc-2610031753359849"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reserve.md"
remedy: "Point both pages at the board and the record page as where a person sees reservations, and name abcd peers as the agents' listing."
resolution: "both pages, and the spec, now name the board and the record page as where a person sees reservations, and abcd peers as an agent's verb"
impact: internal
resolved_by:
  spec: "spc-2610031753359849"
---

The reserve page and its brief chapter say a person sees reservations through abcd peers, but peers is an agent's verb (block: agents, user-invocable: false); a person sees them on the board (bare abcd) and on a record's page (abcd <record-id>), as step 11 of spc-2610031753359849 built.
