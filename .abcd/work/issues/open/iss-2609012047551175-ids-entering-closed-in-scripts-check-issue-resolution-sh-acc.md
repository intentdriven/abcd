---
schema_version: 1
id: "iss-2609012047551175"
slug: "ids-entering-closed-in-scripts-check-issue-resolution-sh-acc"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
---

ids_entering_closed in scripts/check-issue-resolution.sh accepts a rename whose SOURCE is already a terminal folder: it keys on the destination alone, so a record that moves resolved/ to wontfix/, or is reslugged within resolved/, counts as ENTERING a terminal folder. Two topologies let a stale trailer pass silently: main has since moved the record from resolved/ to wontfix/ (the branch's Resolves trailer is satisfied by a move it did not make), and main has reslugged the record inside resolved/ (the rename's destination is terminal, the id is extracted from the new basename, and the trailer is satisfied by a rename). Pre-existing, untouched by the hygiene branch; found by the ruthless review of it. The honest test is that the rename's source is NOT a terminal folder, or that the record was open at the base.
