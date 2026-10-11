---
schema_version: 1
id: "iss-2609012047551175"
slug: "ids-entering-closed-in-scripts-check"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
resolution: "ids_entering_closed now counts an id as entering resolved/ or wontfix/ only when the merge base did not hold it in the very folder it lands in (the merge base's listing, so a move rewritten past rename detection is caught too, and a branch's own wontfix/ -> resolved/ move still enters: a24c792e8); ids_entering_shipped filters on the base's shipped/ listing. Proved by four cases in scripts/check-issue-resolution-cases.sh (base-side resolved->wontfix move, base-side reslug, the move rewritten past rename detection, the shipped/ reslug twin), each passing the old gate and refused by the new one, while a branch's own wontfix/ -> resolved/ move passes."
impact: internal
resolved_by:
  commit: "46ae88404"
---

ids_entering_closed in scripts/check-issue-resolution.sh accepts a rename whose SOURCE is already a terminal folder: it keys on the destination alone, so a record that moves resolved/ to wontfix/, or is reslugged within resolved/, counts as ENTERING a terminal folder. Two topologies let a stale trailer pass silently: main has since moved the record from resolved/ to wontfix/ (the branch's Resolves trailer is satisfied by a move it did not make), and main has reslugged the record inside resolved/ (the rename's destination is terminal, the id is extracted from the new basename, and the trailer is satisfied by a rename). Pre-existing, untouched by the hygiene branch; found by the ruthless review of it. The honest test is that the rename's source is NOT a terminal folder, or that the record was open at the base.

## Grounds

- pursued: a stale Resolves: or Delivers: trailer is no longer satisfied by a base-side move between or within terminal folders; a same-change capture-and-resolve or an honest open->resolved move still passes, and a clean case in the suite failing would show it wrong
