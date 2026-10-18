---
schema_version: 1
id: "iss-2610170309585684"
slug: "the-glossary-term-abcd-development-brief"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "PR7 build of the v0.15.0 deferred glossary findings"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/intent.md"
remedy: "Mark the grill as a design target here, or rewrite the entry around what ships (the planning interview, or ideate's record-grill leg)."
resolution: "The intent glossary entry's usage examples now name the draft itd-10 and the shipped planning interview instead of the grill sub-verb and the superseded itd-27."
impact: internal
---

The glossary term .abcd/development/brief/glossary/core/intent.md gives /abcd:intent grill itd-27 as a working usage example (lines 37-38), but abcd intent --help lists no grill sub-verb, itd-27 was superseded by itd-94, and the glossary README now calls the grill a design target.
