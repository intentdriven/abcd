---
schema_version: 1
id: "iss-2608231339287073"
slug: "genealogy-chart-needs-a-design-rethink"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/timeline.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: what question should the genealogy chart answer, and should its redesign become an intent?"
remedy: "Waits on the genealogy ruling (what question the picture answers): if 'what replaced what': draw supersession as a layered graph along the time axis, edges routed between layers instead of arcs across the lanes; if 'what happened when': drop the arcs, label each lane, and let a crowded day's capsule expand to its members; either way internal/core/site/timeline.go gains a test that no two label boxes overlap on the committed record's export."
---

The genealogy chart is a draft and needs rethinking as a designed picture (future intent seed). Its five lanes over one axis carry real facts — releases, decisions, intents, specs, principles, and issues as a per-day histogram — but the drawing does not yet read: the supersession arcs and their dashed stubs cross the lanes and collide with each other and with the marks they point at (adr-4, adr-8 and adr-18 overprint), a crowded day collapses into a capsule whose count says nothing about what is inside it, the lanes are unlabelled beyond a store name and a total, and the first-commit marker cuts the whole picture without saying why. It reads as five separate strips sharing an axis rather than as one genealogy. The rethink is a design question, not a defect: what should this picture ANSWER, and what shape answers it. Docked folded at the top of the dashboard as of 2026-08-23.

## Remedy grounds (2026-09-29)

- Layered drawing of directed graphs (rank assignment, crossing reduction, then coordinates) is the established layout for lineage: Gansner, Koutsofios, North and Vo, 'A Technique for Drawing Directed Graphs', IEEE TSE 1993 (https://graphviz.org/documentation/TSE93.pdf, checked 2026-09-30); implemented in Go inside timeline.go, not by adding a layout library.
- The overlap test turns the record's overprint finding (adr-4, adr-8 and adr-18) into a detector. Rejected: tuning the arc offsets, which treats the collision without answering the design question the record asks.
