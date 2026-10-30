---
schema_version: 1
id: "iss-2610241549334990"
slug: "a-spec-s-footprint-packages-line-is"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b04 reading iss-2610150512342464"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "In specPackages in internal/core/implement/loop/reservation.go, read each packages entry as its leading path alone. Split on commas only outside parentheses, drop a trailing parenthetical note, and refuse to reserve an entry that still holds whitespace and names nothing in the tree, naming the spec and the entry. Test it with a packages line carrying internal/core/x (new), a parenthetical with a comma inside it, and a prose entry."
resolution: "A build now reserves each packages entry by its leading path: a trailing parenthetical note is dropped and a comma inside one no longer splits the entry. An entry still prose once its note is dropped refuses the start, naming the spec and the entry, and build next excludes its intent at the footprint check. Two of the three open specs that carried such entries now name their paths alone; the third, spc-2609221011151661, is left to the lane rewriting it, and its build is refused naming the entry until then."
impact: fix
---

A spec's Footprint packages line is reserved exactly as written. An entry carrying a note or prose therefore becomes a footprint path with that text in it. A build of spc-2610111627213138 would reserve internal/core/memnote (new). A build of spc-2610120941441282 would reserve internal/surface/cli (the pre-push hook's fixture tests), and its parenthetical containing a comma would split into the two entries scripts/cloud/connectors/claude (new and the host connector). Overlap compares paths segment by segment, and claim.ValidFootprintPath accepts spaces. So such an entry shares no path with the package it means, and the build does not hold that package against another build. Several open specs carry such entries, among them spc-2609221011151661, spc-2610111627213138 and spc-2610120941441282.
