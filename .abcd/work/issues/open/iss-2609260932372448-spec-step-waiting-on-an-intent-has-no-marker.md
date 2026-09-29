---
schema_version: 1
id: "iss-2609260932372448"
slug: "spec-step-waiting-on-an-intent-has-no-marker"
severity: "minor"
category: "process"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-loop1 item 4"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-2609202134338445-one-verb-takes-a-single-intent-from-ready-to-delivered-witho.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (autonomous run A, 2026-09-29), as the record itself routes it: a blocked or after marker on a spec step changes the build loop's first-unlanded-step rule and the ready row's count, which is a product design point."
---

A spec step that waits on another intent has no marker, so the build loop opens a lane on it: spc-2609202134338445's step 3 (the process driver) waits on itd-2609201916056194, and `build` takes the first unlanded step and briefs a lane there, which can only report the dependency. A blocked:/after: marker would change the first-unlanded-step rule and the ready row's count, which is a product design point, not an implementer's; routed to the product thinker.
