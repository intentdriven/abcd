---
schema_version: 1
id: "iss-2608300114359543"
slug: "phase-1-filing-consistency-findings"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "cold-reading Phase 1 adversarial review, 2026-08-30"
found_at: ".abcd/development (brief, intents, roadmap), internal/core/lifeboat/mapping.go"
resolution: "Every listed inconsistency corrected in place; the decision log carries the list."
impact: internal
---

Record consistency findings from the Phase 1 adversarial review: itd-142 and itd-143 attribute the 2026-08-28 rulings to a third role name; itd-143 still describes the not-yet-real marker as pending in a form the same filing ratifies differently; 06-framing.md places a dated preface between the Construal heading and the token, so the section is not statement-first as its own rule and itd-143's projection criterion require; invariant 15 carries an adoption-date marker no sibling carries; 04-scope.md line 24 still counts seven phase docs; Phase 7 and the log say 'Phase 2 planning' where roadmap Phase 2 is a shipped phase; the phases README says phases 1-6 are organised by capability moment; the lifeboat mapping comment counts 23 rows.
