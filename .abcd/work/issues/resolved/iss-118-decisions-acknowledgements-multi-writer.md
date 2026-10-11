---
schema_version: 1
id: "iss-118"
slug: "decisions-acknowledgements-multi-writer"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "post-merge retrospective"
found_at: ".abcd/work/DECISIONS.md"
resolution: "DECISIONS.md and ACKNOWLEDGEMENTS.md gain merge=union in .gitattributes, matching the existing CHANGELOG.md precedent -- both are append-only ledgers of anonymous, dated, order-independent entries with nothing to actually conflict over."
impact: internal
---

DECISIONS.md and ACKNOWLEDGEMENTS.md are multi-writer single files, the merge-hotspot shape that one-writer-per-file rules out. DECISIONS.md was the only textual conflict of the two-programme merge (resolved by hand as a union of both sides). The remedy is a design call per the principle's bounds: per-decision records (full atomicisation, which needs a new id family plus an armed uniqueness detector) or a merge=union gitattribute (smallest fix, legitimate because the ledger's entries are anonymous dated lines that never need identity). ACKNOWLEDGEMENTS.md is the same shape at lower traffic.