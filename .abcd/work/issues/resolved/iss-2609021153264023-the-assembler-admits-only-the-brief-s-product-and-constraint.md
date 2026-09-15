---
schema_version: 1
id: "iss-2609021153264023"
slug: "the-assembler-admits-only-the-brief-s-product-and-constraint"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "final compliance review of the Iteration 2 materials, 2026-09-02"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/reading/include.go"
resolution: "The include table gains four brief-chapter rows of kind brief-section at the positions the product and constraints rows carry: the surfaces, internals and delivery chapters by directory, and the meta chapter by a row whose source is the brief directory and whose match is the exact basename 00-meta.md. The six chapters are named individually because assembler rule 1 forbids naming brief/ whole, the directory containing the glossary, which keeps its own row. The evidence chapter stays excluded and its exclusion rule now states the ground in the table: verdict material, a prior verdict is revision history, the ground the Audit Notes exclusion rests on."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

The assembler admits only the brief's product and constraints chapters where the design documents name the brief's current text as a reading's object

## Grounds

- pursued: we expect admitting the whole brief bar the evidence chapter and the glossary to make the assembler's object what both design documents call brief current text, because the table admitted two chapters of six while naming the same object; a chapter of brief current text absent from a bundle, or the evidence chapter present in one, would show it wrong
