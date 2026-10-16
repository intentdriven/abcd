---
id: dcn-2607080000000011
slug: corpus-restructured-to-class-segregated
date: 2026-07-08
---
2026-07-08 — Corpus restructured to class-segregated per-source folders
(confidential/<key>/, public/<key>/): confidentiality is declared at
ingestion and LOCATION is its single source of truth (flag mirrors, tooling
refuses on mismatch); derived artifacts inherit by location; declassification
is a visible git mv.
