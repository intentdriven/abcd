---
schema_version: 1
id: "iss-284"
slug: "the-adr-40-vocabulary-renames-narrow-the"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "v0.6.0 cut"
resolution: "the renames shipped with itd-123/124/125; this record declares the surface narrowing for the cut"
impact: breaking
---

The adr-40 vocabulary renames narrow the public surface for v0.6.0: abcd audit is abcd lint, abcd disembark oracle is abcd disembark review, and abcd intent review / intent review ingest are abcd intent audit / intent audit ingest. Four command paths leave the surface; each successor refuses the old spelling by naming the new one. Deliberate breaks, documented in the CHANGELOG Breaking section (itd-123, itd-124, itd-125)