---
schema_version: 1
id: "iss-2609260935465554"
slug: "the-brief-s-context-page-01-product-02-context-md-the-abcd"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md"
resolution: "The context page lists consistency with the intent sub-verbs that ship."
impact: internal
resolved_by:
  commit: "d589faca"
---

The brief's context page (01-product/02-context.md, the /abcd:intent bullet) lists consistency among the /abcd:intent sub-verbs remaining design targets, while itd-48 has shipped it and the press release says it ships.

## Grounds

- pursued: the brief names consistency as shipped wherever it lists the intent sub-verbs; shown wrong if a brief page still lists it as a design target
